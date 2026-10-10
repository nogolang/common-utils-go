package payUtils

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/alipay"

	"github.com/nogolang/common-utils-go/payUtils/internal/money"
)

var _ PayChannel = (*AlipayChannel)(nil)

// AlipayChannel 支付宝渠道实现，只负责跟支付宝打交道。
type AlipayChannel struct {
	sdk    alipaySDK
	config AlipayConfig
}

// NewAlipayChannel validates configuration and creates a reusable Alipay adapter.
func NewAlipayChannel(config AlipayConfig) (*AlipayChannel, error) {
	if config.AppID == "" || config.AppPrivateKey == "" || config.PublicKey == "" || config.NotifyURL == "" {
		return nil, fmt.Errorf("alipay configuration is incomplete")
	}
	client, err := alipay.NewClient(config.AppID, config.AppPrivateKey, config.IsProduction)
	if err != nil {
		return nil, fmt.Errorf("initialize alipay client: %w", err)
	}
	client.DebugSwitch = gopay.DebugOff
	return &AlipayChannel{sdk: client, config: config}, nil
}

func (channel *AlipayChannel) Name() string { return Alipay }

func (channel *AlipayChannel) PayModes() []string { return []string{PayModeQRCode} }

// CreateOrder 统一收单线下交易预创建（当面付），返回二维码内容。
func (channel *AlipayChannel) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResult, error) {
	if req == nil {
		return nil, fmt.Errorf("alipay create order request is nil")
	}
	if req.PayMode != PayModeQRCode {
		return nil, fmt.Errorf("%w: %s/%s", ErrPayModeUnsupported, Alipay, req.PayMode)
	}
	subject := req.Subject
	if subject == "" {
		subject = fmt.Sprintf("order_%s", req.OutTradeNo)
	}

	// 初始化 请求参数
	bm := make(gopay.BodyMap)
	bm.
		//订单标题，必填
		Set("subject", subject).
		//商户订单号
		Set("out_trade_no", req.OutTradeNo).
		//订单总金额，支付宝的单位是元，本地账本是分，这里换算
		Set("total_amount", money.CentsToYuan(req.AmountCents)).
		//相对过期时间，service 已经算好了 ExpireAt，这里只是换成支付宝要的 "5m" 形式
		Set("timeout_express", fmt.Sprintf("%dm", req.ExpireSeconds/60)).
		//异步回调地址；stateVersion 等业务信息不能由商户在这里追加
		Set("notify_url", channel.config.NotifyURL)

	//TradePrecreate在文档里是 "统一收单线下交易预创建"。
	//大部分要扫码的，都是这个接口。
	//product_code固定QR_CODE_OFFLINE，但传了会报错，最好别传。
	//https://opendocs.alipay.com/open-v3/08c7f9f8_alipay.trade.pay?scene=32&pathHash=86db8e4a
	rsp, err := channel.sdk.TradePrecreate(ctx, bm)
	if err != nil {
		return nil, err
	}
	if rsp == nil || rsp.Response == nil {
		return nil, fmt.Errorf("支付宝下单响应为空")
	}
	return &CreateOrderResult{
		Channel:           Alipay,
		PayMode:           req.PayMode,
		OutTradeNo:        req.OutTradeNo,
		ExpireSeconds:     req.ExpireSeconds,
		CodeURL:           rsp.Response.QrCode,
		PayPayload:        rsp.Response.QrCode,
		PendingTradeState: "WAIT_BUYER_PAY",
		Raw:               rsp.Response,
	}, nil
}

// QueryOrder 查询订单。
func (channel *AlipayChannel) QueryOrder(ctx context.Context, outTradeNo string) (*OrderQueryResult, error) {
	bm := make(gopay.BodyMap)
	// 初始化 请求参数
	bm.Set("out_trade_no", outTradeNo)
	rsp, err := channel.sdk.TradeQuery(ctx, bm)
	if err != nil {
		return nil, fmt.Errorf("查询订单失败:%w", err)
	}
	if rsp == nil || rsp.Response == nil {
		return nil, fmt.Errorf("支付宝查单响应为空")
	}
	trade := rsp.Response
	amountCents, err := money.YuanToCents(trade.TotalAmount)
	if err != nil {
		return nil, fmt.Errorf("支付宝订单金额格式无效: %w", err)
	}
	return &OrderQueryResult{
		Channel:            Alipay,
		OutTradeNo:         firstNonEmpty(trade.OutTradeNo, outTradeNo),
		Status:             normalizeAlipayTradeState(trade.TradeStatus),
		PlatformTradeState: trade.TradeStatus,
		TransactionID:      trade.TradeNo,
		AmountCents:        amountCents,
		PaidAt:             parseAlipayTime(trade.SendPayDate),
		Raw:                trade,
	}, nil
}

// Refund 申请退款。
func (channel *AlipayChannel) Refund(ctx context.Context, req *RefundRequest) (*RefundResult, error) {
	if req == nil {
		return nil, fmt.Errorf("alipay refund request is nil")
	}
	reason := req.Reason
	if reason == "" {
		reason = "支付演示退款"
	}
	bm := make(gopay.BodyMap)
	// 初始化 请求参数
	bm.
		//商户订单号
		Set("out_trade_no", req.OutTradeNo).
		//退款单号，支付宝侧的幂等键：同一个 out_request_no 重复请求只会真退一次
		Set("out_request_no", req.OutRefundNo).
		//退款金额，单位元
		Set("refund_amount", money.CentsToYuan(req.RefundAmountCents)).
		//退款原因，会在商户和用户pc的退款订单中展示
		Set("refund_reason", reason)

	//同一笔交易的退款至少间隔3s后发起。
	//退款成功判断说明：接口返回 fund_change=Y 为退款成功，
	//fund_change=N 或无此字段值返回时需通过退款查询接口进一步确认退款状态。
	//注意，接口中 code=10000，仅代表本次退款请求成功，不代表退款成功。
	//退款商品列表信息
	//bm.Set("refund_goods_detail", []map[string]interface{}{})
	//当添加了query_options，此时TradeRefund会返回额外的信息。
	//deposit_back_info如果存在退到银行卡的场景，此时等待银行收到钱的回执消息才会响应
	//此时会返回一个has_deposit_back，它是true
	//如果是普通场景，它则为false
	//bm.Set("query_options", []string{"deposit_back_info"})
	rsp, err := channel.sdk.TradeRefund(ctx, bm)
	if err != nil {
		return nil, err
	}
	if rsp == nil || rsp.Response == nil {
		return nil, fmt.Errorf("支付宝退款响应为空")
	}
	//必须要格外注意，重复退款是会显示退款成功的，但是金额肯定不会退回来；
	//所以在演示代码中要通过 FundChange 判断是否真的发生金额变动。
	refund := rsp.Response
	state := "PROCESSING"
	var successAt *time.Time
	if refund.GmtRefundPay != "" || refund.FundChange == "Y" {
		state = "SUCCESS"
		successAt = parseAlipayTime(refund.GmtRefundPay)
	}
	return &RefundResult{
		Channel:           Alipay,
		OutTradeNo:        req.OutTradeNo,
		OutRefundNo:       req.OutRefundNo,
		Status:            state,
		PlatformState:     refund.FundChange,
		RefundAmountCents: req.RefundAmountCents,
		TotalAmountCents:  req.TotalAmountCents,
		SuccessAt:         successAt,
		Raw:               refund,
	}, nil
}

// QueryRefund 查询订单退款结果。
func (channel *AlipayChannel) QueryRefund(ctx context.Context, outTradeNo, outRefundNo string) (*RefundResult, error) {
	bm := make(gopay.BodyMap)
	// 初始化 请求参数
	bm.Set("out_request_no", outRefundNo).
		Set("out_trade_no", outTradeNo).
		//填写了query_options，会返回额外的信息。
		//refund_detail_item_list本次退款使用的资金渠道
		//gmt_refund_pay退款执行成功的时间
		//deposit_back_info 银行卡到账时间，如果你是用银行卡付款的话
		Set("query_options", []string{"refund_detail_item_list", "gmt_refund_pay", "deposit_back_info"})
	rsp, err := channel.sdk.TradeFastPayRefundQuery(ctx, bm)
	if err != nil {
		return nil, fmt.Errorf("查询订单失败:%w", err)
	}
	if rsp == nil || rsp.Response == nil {
		return nil, fmt.Errorf("支付宝退款查询响应为空")
	}
	refund := rsp.Response
	amountCents, err := money.YuanToCents(refund.RefundAmount)
	if err != nil {
		return nil, fmt.Errorf("支付宝退款金额格式无效: %w", err)
	}
	return &RefundResult{
		Channel:           Alipay,
		OutTradeNo:        firstNonEmpty(refund.OutTradeNo, outTradeNo),
		OutRefundNo:       firstNonEmpty(refund.OutRequestNo, outRefundNo),
		Status:            normalizeAlipayRefundState(refund.RefundStatus),
		PlatformState:     refund.RefundStatus,
		RefundAmountCents: amountCents,
		SuccessAt:         parseAlipayTime(refund.GmtRefundPay),
		Raw:               refund,
	}, nil
}

/*
支付宝回调如果失败，会立即重试3次，然后按照这个间隔去重试，4m、10m、10m、1h、2h、6h、15h
如果还没有通知到，就会被标记为失败，此时我们只能主动查单
因为不是最大努力通知，所以我们一定要有一个查询订单的线程
*/
func (channel *AlipayChannel) HandleCallback(
	ctx context.Context,
	request *http.Request,
	meta CallbackMeta,
) (*CallbackResult, error) {
	if request == nil {
		return nil, fmt.Errorf("alipay callback request is nil")
	}
	body, err := io.ReadAll(request.Body)
	if err != nil {
		slog.ErrorContext(ctx, "读取支付宝回调失败", slog.String("error", err.Error()))
		return nil, err
	}
	//设置回去，后续解析器还要读取 body。
	request.Body = io.NopCloser(bytes.NewBuffer(body))
	//解析出 body，校验签名。
	notifyBody, err := alipay.ParseNotifyToBodyMap(request)
	if err != nil {
		slog.ErrorContext(ctx, "解析支付宝回调失败", slog.String("error", err.Error()))
		return nil, err
	}

	orderNo := notifyBody.Get("out_trade_no")
	//传入支付宝返回的公钥验签，确认回调请求有效。
	//支付宝回调请求需要验签，避免把伪造请求当成支付通知。
	if _, err := alipay.VerifySign(channel.config.PublicKey, notifyBody); err != nil {
		slog.WarnContext(ctx, "支付宝回调验签失败",
			slog.String("ip", meta.RealIP),
			slog.String("order_no", orderNo),
			slog.String("error", err.Error()))
		return nil, err
	}

	//这里不需要判断，因为只有支付成功才会触发回调。
	//https://opendocs.alipay.com/open/0c2c19?pathHash=df12a335
	tradeStatus := notifyBody.Get("trade_status")
	if tradeStatus != "TRADE_SUCCESS" && tradeStatus != "TRADE_FINISHED" {
		return nil, fmt.Errorf("支付宝回调交易状态不是成功: %s", tradeStatus)
	}

	//只要是相同的调用 notify_id 都是相同的，它是一个长字符串。
	slog.InfoContext(ctx, "支付宝支付回调接受",
		slog.String("order_no", orderNo),
		slog.String("notify_id", notifyBody.Get("notify_id")))
	//下面是支付成功的业务。
	slog.InfoContext(ctx, "支付宝支付成功", slog.String("order_no", orderNo))
	amountCents, _ := money.YuanToCents(notifyBody.Get("total_amount"))
	return &CallbackResult{
		Channel:        Alipay,
		OutTradeNo:     orderNo,
		TransactionID:  notifyBody.Get("trade_no"),
		TradeState:     tradeStatus,
		Status:         normalizeAlipayTradeState(tradeStatus),
		AmountCents:    amountCents,
		PaidAt:         parseAlipayTime(notifyBody.Get("gmt_payment")),
		AckStatus:      http.StatusOK,
		AckContentType: "text/plain; charset=utf-8",
		AckBody:        "success",
	}, nil
}

// FailureAck 支付宝只认纯文本应答：只要不是 success，都算失败并会按 4m/10m/10m/1h... 重试。
func (channel *AlipayChannel) FailureAck(err error) *CallbackResult {
	return &CallbackResult{
		Channel:        Alipay,
		AckStatus:      http.StatusOK,
		AckContentType: "text/plain; charset=utf-8",
		AckBody:        "fail",
	}
}

// normalizeAlipayTradeState 平台交易状态 → 跨渠道归一化状态，前端轮询只认这一套。
func normalizeAlipayTradeState(state string) string {
	switch state {
	case "TRADE_SUCCESS", "TRADE_FINISHED":
		return "SUCCESS"
	case "WAIT_BUYER_PAY":
		return "NOTPAY"
	case "TRADE_CLOSED":
		return "CLOSED"
	default:
		return "UNKNOWN"
	}
}

func normalizeAlipayRefundState(state string) string {
	switch state {
	case "REFUND_SUCCESS":
		return "SUCCESS"
	case "REFUND_PROCESSING":
		return "PROCESSING"
	case "REFUND_CLOSED":
		return "CLOSED"
	default:
		return "UNKNOWN"
	}
}

// parseAlipayTime 支付宝的时间是 "2006-01-02 15:04:05" 且不带时区，按本地时区解析。
func parseAlipayTime(value string) *time.Time {
	if value == "" {
		return nil
	}
	parsed, err := time.ParseInLocation("2006-01-02 15:04:05", value, time.Local)
	if err != nil {
		return nil
	}
	return &parsed
}
