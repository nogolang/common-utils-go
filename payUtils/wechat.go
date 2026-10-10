package payUtils

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-pay/gopay"
	wechatv3 "github.com/go-pay/gopay/wechat/v3"
)

var _ PayChannel = (*WechatChannel)(nil)

// WechatChannel 微信支付渠道实现。
type WechatChannel struct {
	client *wechatv3.ClientV3
	sdk    wechatSDK
	config WechatConfig
}

// NewWechatChannel validates merchant material and creates a reusable WeChat adapter.
func NewWechatChannel(config WechatConfig) (*WechatChannel, error) {
	if config.AppID == "" || config.MchID == "" || config.SerialNo == "" || config.PrivateKey == "" || config.APIV3Key == "" || config.PayPublicKeyID == "" || config.PayPublicKeyPEM == "" {
		return nil, fmt.Errorf("wechat configuration is incomplete")
	}
	if config.NativeNotifyURL == "" || config.JSAPINotifyURL == "" || config.H5NotifyURL == "" {
		return nil, fmt.Errorf("wechat notify URLs are incomplete")
	}
	if err := validateWechatMerchantMaterial(config); err != nil {
		return nil, err
	}
	client, err := wechatv3.NewClientV3(config.MchID, config.SerialNo, config.APIV3Key, config.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("initialize wechat v3 client: %w", err)
	}
	if err := client.AutoVerifySignByPublicKey([]byte(config.PayPublicKeyPEM), config.PayPublicKeyID); err != nil {
		return nil, fmt.Errorf("configure wechat callback signature verification: %w", err)
	}
	client.DebugSwitch = gopay.DebugOff
	return &WechatChannel{client: client, sdk: client, config: config}, nil
}

func validateWechatMerchantMaterial(config WechatConfig) error {
	certBlock, _ := pem.Decode([]byte(config.MchCertPEM))
	if certBlock == nil {
		return fmt.Errorf("parse wechat merchant certificate: invalid PEM")
	}
	certificate, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return fmt.Errorf("parse wechat merchant certificate: %w", err)
	}
	keyBlock, _ := pem.Decode([]byte(config.PrivateKey))
	if keyBlock == nil {
		return fmt.Errorf("parse wechat merchant private key: invalid PEM")
	}
	parsedKey, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	if err != nil {
		return fmt.Errorf("parse wechat merchant private key: %w", err)
	}
	rsaPrivateKey, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		return fmt.Errorf("wechat merchant private key is not RSA")
	}
	certPublicKey, ok := certificate.PublicKey.(*rsa.PublicKey)
	if !ok || !rsaPrivateKey.PublicKey.Equal(certPublicKey) {
		return fmt.Errorf("wechat merchant certificate and private key do not match")
	}
	if !strings.EqualFold(fmt.Sprintf("%X", certificate.SerialNumber), config.SerialNo) {
		return fmt.Errorf("wechat merchant certificate serial number does not match")
	}
	return nil
}

func (channel *WechatChannel) Name() string { return Wechat }

func (channel *WechatChannel) PayModes() []string {
	return []string{PayModeNative, PayModeJSAPI, PayModeH5}
}

// CreateOrder 按支付方式分发到 Native / JSAPI / H5。
func (channel *WechatChannel) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResult, error) {
	if req == nil {
		return nil, fmt.Errorf("wechat create order request is nil")
	}
	switch req.PayMode {
	case PayModeNative:
		return channel.createNative(ctx, req)
	case PayModeJSAPI:
		return channel.createJSAPI(ctx, req)
	case PayModeH5:
		return channel.createH5(ctx, req)
	default:
		return nil, fmt.Errorf("%w: %s/%s", ErrPayModeUnsupported, Wechat, req.PayMode)
	}
}

// createNative Native 下单，返回 weixin:// 开头的二维码内容。
func (channel *WechatChannel) createNative(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResult, error) {
	bm := make(gopay.BodyMap)
	bm.
		Set("appid", channel.config.AppID).
		Set("mchid", channel.config.MchID).
		Set("description", wechatSubject(req.Subject, "微信Native支付演示")).
		Set("out_trade_no", req.OutTradeNo).
		Set("time_expire", req.ExpireAt.Format(time.RFC3339)).
		Set("notify_url", channel.config.NativeNotifyURL).
		SetBodyMap("amount", func(bm gopay.BodyMap) {
			//微信的金额单位就是分，不用换算
			bm.Set("total", req.AmountCents).Set("currency", "CNY")
		})

	rsp, err := channel.sdk.V3TransactionNative(ctx, bm)
	if err != nil {
		return nil, err
	}
	if rsp == nil {
		return nil, fmt.Errorf("微信Native下单响应为空")
	}
	if rsp.Code != wechatv3.Success {
		return nil, wechatProviderError(rsp.Code, rsp.Error, rsp.ErrResponse)
	}
	if rsp.Response == nil {
		return nil, fmt.Errorf("微信Native下单响应为空")
	}
	return &CreateOrderResult{
		Channel:           Wechat,
		PayMode:           req.PayMode,
		OutTradeNo:        req.OutTradeNo,
		ExpireSeconds:     req.ExpireSeconds,
		CodeURL:           rsp.Response.CodeUrl,
		PayPayload:        rsp.Response.CodeUrl,
		PendingTradeState: "NOTPAY",
		Raw:               rsp.Response,
	}, nil
}

// createJSAPI 小程序 / 公众号下单，返回 uni.requestPayment 需要的签名参数。
func (channel *WechatChannel) createJSAPI(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResult, error) {
	if req.OpenID == "" {
		return nil, fmt.Errorf("wechat JSAPI openid is required")
	}
	bm := make(gopay.BodyMap)
	bm.
		Set("appid", channel.config.AppID).
		Set("mchid", channel.config.MchID).
		Set("description", wechatSubject(req.Subject, "微信JSAPI支付演示")).
		Set("out_trade_no", req.OutTradeNo).
		Set("time_expire", req.ExpireAt.Format(time.RFC3339)).
		Set("notify_url", channel.config.JSAPINotifyURL).
		SetBodyMap("amount", func(bm gopay.BodyMap) {
			bm.Set("total", req.AmountCents).Set("currency", "CNY")
		}).
		SetBodyMap("payer", func(bm gopay.BodyMap) {
			bm.Set("openid", req.OpenID)
		})

	rsp, err := channel.sdk.V3TransactionJsapi(ctx, bm)
	if err != nil {
		return nil, err
	}
	if rsp == nil {
		return nil, fmt.Errorf("微信JSAPI下单响应为空")
	}
	if rsp.Code != wechatv3.Success {
		return nil, wechatProviderError(rsp.Code, rsp.Error, rsp.ErrResponse)
	}
	if rsp.Response == nil {
		return nil, fmt.Errorf("微信JSAPI下单响应为空")
	}
	//拿到 prepay_id 之后还要再用商户私钥签一次名，小程序才认；
	//这一步失败就没法拉起支付，所以直接当下单失败返回。
	payParams, err := channel.sdk.PaySignOfJSAPI(channel.config.AppID, rsp.Response.PrepayId)
	if err != nil {
		return nil, err
	}
	return &CreateOrderResult{
		Channel:           Wechat,
		PayMode:           req.PayMode,
		OutTradeNo:        req.OutTradeNo,
		ExpireSeconds:     req.ExpireSeconds,
		JSAPIParams:       newJSAPIParams(payParams),
		PayPayload:        rsp.Response.PrepayId,
		PendingTradeState: "NOTPAY",
		Raw:               rsp.Response,
	}, nil
}

// createH5 H5 下单，返回拉起微信收银台的中间页链接。
func (channel *WechatChannel) createH5(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResult, error) {
	if req.PayerClientIP == "" {
		return nil, fmt.Errorf("wechat H5 payer client IP is required")
	}
	bm := make(gopay.BodyMap)
	bm.
		Set("appid", channel.config.AppID).
		Set("mchid", channel.config.MchID).
		Set("description", wechatSubject(req.Subject, "微信H5支付演示")).
		Set("out_trade_no", req.OutTradeNo).
		Set("time_expire", req.ExpireAt.Format(time.RFC3339)).
		Set("notify_url", channel.config.H5NotifyURL).
		SetBodyMap("amount", func(bm gopay.BodyMap) {
			bm.Set("total", req.AmountCents).Set("currency", "CNY")
		}).
		SetBodyMap("scene_info", func(bm gopay.BodyMap) {
			//H5 支付必须带用户真实 IP，否则微信会拒单
			bm.
				Set("payer_client_ip", req.PayerClientIP).
				SetBodyMap("h5_info", func(bm gopay.BodyMap) {
					bm.Set("type", "Wap")
				})
		})

	rsp, err := channel.sdk.V3TransactionH5(ctx, bm)
	if err != nil {
		return nil, err
	}
	if rsp == nil {
		return nil, fmt.Errorf("微信H5下单响应为空")
	}
	if rsp.Code != wechatv3.Success {
		return nil, wechatProviderError(rsp.Code, rsp.Error, rsp.ErrResponse)
	}
	if rsp.Response == nil {
		return nil, fmt.Errorf("微信H5下单响应为空")
	}
	return &CreateOrderResult{
		Channel:           Wechat,
		PayMode:           req.PayMode,
		OutTradeNo:        req.OutTradeNo,
		ExpireSeconds:     req.ExpireSeconds,
		H5URL:             rsp.Response.H5Url,
		PayPayload:        rsp.Response.H5Url,
		PendingTradeState: "NOTPAY",
		Raw:               rsp.Response,
	}, nil
}

func (channel *WechatChannel) QueryOrder(ctx context.Context, outTradeNo string) (*OrderQueryResult, error) {
	rsp, err := channel.sdk.V3TransactionQueryOrder(ctx, wechatv3.OutTradeNo, outTradeNo)
	if err != nil {
		return nil, err
	}
	if rsp == nil {
		return nil, fmt.Errorf("微信查单响应为空")
	}
	if rsp.Code != wechatv3.Success {
		return nil, wechatProviderError(rsp.Code, rsp.Error, rsp.ErrResponse)
	}
	if rsp.Response == nil {
		return nil, fmt.Errorf("微信查单响应为空")
	}
	trade := rsp.Response
	var amountCents int64
	if trade.Amount != nil {
		amountCents = int64(trade.Amount.Total)
	}
	return &OrderQueryResult{
		Channel:            Wechat,
		OutTradeNo:         firstNonEmpty(trade.OutTradeNo, outTradeNo),
		Status:             normalizeWechatTradeState(trade.TradeState),
		PlatformTradeState: trade.TradeState,
		TransactionID:      trade.TransactionId,
		AmountCents:        amountCents,
		PaidAt:             parseWechatTime(trade.SuccessTime),
		Raw:                trade,
	}, nil
}

func (channel *WechatChannel) Refund(ctx context.Context, req *RefundRequest) (*RefundResult, error) {
	if req == nil {
		return nil, fmt.Errorf("wechat refund request is nil")
	}
	bm := make(gopay.BodyMap)
	bm.
		Set("out_trade_no", req.OutTradeNo).
		Set("out_refund_no", req.OutRefundNo).
		SetBodyMap("amount", func(bm gopay.BodyMap) {
			//微信退款必须同时传 refund 和 total，total 要等于订单实付金额，
			//否则部分退款会直接被拒
			bm.
				Set("refund", req.RefundAmountCents).
				Set("total", req.TotalAmountCents).
				Set("currency", "CNY")
		})
	if req.Reason != "" {
		bm.Set("reason", req.Reason)
	}
	rsp, err := channel.sdk.V3Refund(ctx, bm)
	if err != nil {
		return nil, err
	}
	if rsp == nil {
		return nil, fmt.Errorf("微信退款响应为空")
	}
	if rsp.Code != wechatv3.Success {
		return nil, wechatProviderError(rsp.Code, rsp.Error, rsp.ErrResponse)
	}
	if rsp.Response == nil {
		return nil, fmt.Errorf("微信退款响应为空")
	}
	refund := rsp.Response
	platform := refund.Status
	amountCents := req.RefundAmountCents
	if refund.Amount != nil {
		amountCents = int64(refund.Amount.Refund)
	}
	return &RefundResult{
		Channel:           Wechat,
		OutTradeNo:        firstNonEmpty(refund.OutTradeNo, req.OutTradeNo),
		OutRefundNo:       firstNonEmpty(refund.OutRefundNo, req.OutRefundNo),
		RefundID:          refund.RefundId,
		Status:            normalizeWechatRefundState(platform),
		PlatformState:     platform,
		RefundAmountCents: amountCents,
		TotalAmountCents:  req.TotalAmountCents,
		SuccessAt:         parseWechatTime(refund.SuccessTime),
		Raw:               refund,
	}, nil
}

func (channel *WechatChannel) QueryRefund(ctx context.Context, outTradeNo, outRefundNo string) (*RefundResult, error) {
	rsp, err := channel.sdk.V3RefundQuery(ctx, outRefundNo, nil)
	if err != nil {
		return nil, err
	}
	if rsp == nil {
		return nil, fmt.Errorf("微信退款查询响应为空")
	}
	if rsp.Code != wechatv3.Success {
		return nil, wechatProviderError(rsp.Code, rsp.Error, rsp.ErrResponse)
	}
	if rsp.Response == nil {
		return nil, fmt.Errorf("微信退款查询响应为空")
	}
	refund := rsp.Response
	var refundAmount, totalAmount int64
	if refund.Amount != nil {
		refundAmount = int64(refund.Amount.Refund)
		totalAmount = int64(refund.Amount.Total)
	}
	return &RefundResult{
		Channel:           Wechat,
		OutTradeNo:        firstNonEmpty(refund.OutTradeNo, outTradeNo),
		OutRefundNo:       firstNonEmpty(refund.OutRefundNo, outRefundNo),
		RefundID:          refund.RefundId,
		Status:            normalizeWechatRefundState(refund.Status),
		PlatformState:     refund.Status,
		RefundAmountCents: refundAmount,
		TotalAmountCents:  totalAmount,
		SuccessAt:         parseWechatTime(refund.SuccessTime),
		Raw:               refund,
	}, nil
}

// HandleCallback 解析、验签、解密微信异步通知。
//
// 微信回调比支付宝多两步：先用微信支付公钥验签，再用 APIv3 密钥（AES-256-GCM）
// 解密 resource 里的密文，明文里才是真正的订单结果。
func (channel *WechatChannel) HandleCallback(
	ctx context.Context,
	request *http.Request,
	meta CallbackMeta,
) (*CallbackResult, error) {
	if request == nil {
		return nil, fmt.Errorf("wechat callback request is nil")
	}
	notifyReq, err := wechatv3.V3ParseNotify(request)
	if err != nil {
		return nil, err
	}
	if err := notifyReq.VerifySignByPK(channel.client.WxPublicKey()); err != nil {
		err = fmt.Errorf("签名校验失败: %w", err)
		slog.WarnContext(ctx, "微信回调验签失败",
			slog.String("ip", meta.RealIP),
			slog.String("error", err.Error()))
		return nil, err
	}
	payResult, err := notifyReq.DecryptPayCipherText(string(channel.client.ApiV3Key))
	if err != nil {
		return nil, err
	}
	if payResult.TradeState != wechatv3.TradeStateSuccess {
		return nil, fmt.Errorf("订单未支付成功")
	}
	slog.InfoContext(ctx, "微信支付成功",
		slog.String("order_no", payResult.OutTradeNo),
		slog.String("transaction_id", payResult.TransactionId))
	var amountCents int64
	if payResult.Amount != nil {
		amountCents = int64(payResult.Amount.Total)
	}
	return &CallbackResult{
		Channel:        Wechat,
		OutTradeNo:     payResult.OutTradeNo,
		TransactionID:  payResult.TransactionId,
		TradeState:     payResult.TradeState,
		Status:         normalizeWechatTradeState(payResult.TradeState),
		AmountCents:    amountCents,
		PaidAt:         parseWechatTime(payResult.SuccessTime),
		AckStatus:      http.StatusOK,
		AckContentType: "application/json; charset=utf-8",
		AckBody:        wechatAckBody("SUCCESS", "成功"),
	}, nil
}

// FailureAck 微信要求失败时回 HTTP 5xx + {"code":"FAIL"}，它才会重试。
func (channel *WechatChannel) FailureAck(err error) *CallbackResult {
	message := ""
	if err != nil {
		message = err.Error()
	}
	return &CallbackResult{
		Channel:        Wechat,
		AckStatus:      http.StatusInternalServerError,
		AckContentType: "application/json; charset=utf-8",
		AckBody:        wechatAckBody("FAIL", message),
	}
}

// wechatAckBody 用 gin.H 同款的 map 序列化，保证 key 顺序和原来 c.JSON 的输出逐字一致。
func wechatAckBody(code, message string) string {
	payload, err := json.Marshal(map[string]any{"code": code, "message": message})
	if err != nil {
		return `{"code":"FAIL","message":"serialize ack failed"}`
	}
	return string(payload)
}

// wechatProviderError 把微信的业务错误码兜成 ProviderError，control 会把 detail 一起回给前端。
func wechatProviderError(code int, message string, detail any) error {
	if message == "" {
		message = fmt.Sprintf("微信支付接口返回错误码 %d", code)
	}
	return &ProviderError{Message: message, Data: detail}
}

func wechatSubject(subject, fallback string) string {
	if subject == "" {
		return fallback
	}
	return subject
}

// newJSAPIParams converts gopay's JSAPI signing fields to the public payment result.
func newJSAPIParams(params *wechatv3.JSAPIPayParams) *JSAPIParams {
	if params == nil {
		return nil
	}
	return &JSAPIParams{
		AppID:     params.AppId,
		TimeStamp: params.TimeStamp,
		NonceStr:  params.NonceStr,
		Package:   params.Package,
		SignType:  params.SignType,
		PaySign:   params.PaySign,
	}
}

// normalizeWechatTradeState 平台交易状态 → 跨渠道归一化状态。
func normalizeWechatTradeState(state string) string {
	switch state {
	case wechatv3.TradeStateSuccess:
		return "SUCCESS"
	case wechatv3.TradeStateNoPay, wechatv3.TradeStatePaying:
		return "NOTPAY"
	case wechatv3.TradeStateClosed, wechatv3.TradeStateRevoked:
		return "CLOSED"
	case wechatv3.TradeStateRefund:
		return "REFUND"
	case wechatv3.TradeStatePayError:
		return "FAILED"
	default:
		return "UNKNOWN"
	}
}

func normalizeWechatRefundState(state string) string {
	switch state {
	case "SUCCESS":
		return "SUCCESS"
	case "PROCESSING":
		return "PROCESSING"
	case "CLOSED":
		return "CLOSED"
	case "ABNORMAL":
		return "FAILED"
	default:
		return "UNKNOWN"
	}
}

// parseWechatTime 微信的时间是 RFC3339 带时区，直接解析。
func parseWechatTime(value string) *time.Time {
	if value == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil
	}
	return &parsed
}
