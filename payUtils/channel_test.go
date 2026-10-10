package payUtils

import (
	"context"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-pay/gopay"
	alipaySDKPkg "github.com/go-pay/gopay/alipay"
	"github.com/go-pay/gopay/wechat/v3"
)

func TestConstructorsRejectIncompleteConfiguration(t *testing.T) {
	if _, err := NewAlipayChannel(AlipayConfig{}); err == nil {
		t.Fatal("NewAlipayChannel accepted an empty configuration")
	}
	if _, err := NewWechatChannel(WechatConfig{}); err == nil {
		t.Fatal("NewWechatChannel accepted an empty configuration")
	}
}

func TestRegistryLookupAndPaymentModes(t *testing.T) {
	alipay := &testChannel{name: Alipay, modes: []string{PayModeQRCode}}
	registry := NewRegistry(alipay)
	got, err := registry.Get("  ALIPAY ")
	if err != nil || got != alipay {
		t.Fatalf("Get() = (%v, %v), want registered channel", got, err)
	}
	if !Supports(alipay, PayModeQRCode) || Supports(alipay, PayModeJSAPI) {
		t.Fatal("Supports returned incorrect payment mode results")
	}
	if _, err := registry.Get("missing"); !errors.Is(err, ErrUnknownChannel) {
		t.Fatalf("Get(missing) error = %v, want ErrUnknownChannel", err)
	}
}

func TestUnsupportedModeDoesNotCallSDK(t *testing.T) {
	if _, err := (&AlipayChannel{}).CreateOrder(context.Background(), &CreateOrderRequest{PayMode: PayModeJSAPI}); !errors.Is(err, ErrPayModeUnsupported) {
		t.Fatalf("alipay CreateOrder error = %v", err)
	}
	if _, err := (&WechatChannel{}).CreateOrder(context.Background(), &CreateOrderRequest{PayMode: PayModeQRCode}); !errors.Is(err, ErrPayModeUnsupported) {
		t.Fatalf("wechat CreateOrder error = %v", err)
	}
}

func TestAlipayPaymentOperations(t *testing.T) {
	fake := &fakeAlipaySDK{}
	channel := &AlipayChannel{sdk: fake}
	created, err := channel.CreateOrder(context.Background(), &CreateOrderRequest{PayMode: PayModeQRCode, OutTradeNo: "order-1", AmountCents: 1299, ExpireSeconds: 600})
	if err != nil || created.CodeURL != "alipay://qr" || created.Channel != Alipay {
		t.Fatalf("CreateOrder() = (%+v, %v)", created, err)
	}
	queried, err := channel.QueryOrder(context.Background(), "order-1")
	if err != nil || queried.Status != "SUCCESS" || queried.AmountCents != 1299 {
		t.Fatalf("QueryOrder() = (%+v, %v)", queried, err)
	}
	refunded, err := channel.Refund(context.Background(), &RefundRequest{OutTradeNo: "order-1", OutRefundNo: "refund-1", RefundAmountCents: 300, TotalAmountCents: 1299})
	if err != nil || refunded.Status != "SUCCESS" || refunded.RefundAmountCents != 300 {
		t.Fatalf("Refund() = (%+v, %v)", refunded, err)
	}
	refundQuery, err := channel.QueryRefund(context.Background(), "order-1", "refund-1")
	if err != nil || refundQuery.Status != "SUCCESS" || refundQuery.RefundAmountCents != 300 {
		t.Fatalf("QueryRefund() = (%+v, %v)", refundQuery, err)
	}
	if fake.createCalls != 1 || fake.queryCalls != 1 || fake.refundCalls != 1 || fake.refundQueryCalls != 1 {
		t.Fatalf("unexpected SDK call counts: %+v", fake)
	}
}

func TestWechatPaymentOperations(t *testing.T) {
	fake := &fakeWechatSDK{}
	channel := &WechatChannel{sdk: fake, config: WechatConfig{AppID: "app-1"}}
	for _, mode := range []string{PayModeNative, PayModeJSAPI, PayModeH5} {
		request := &CreateOrderRequest{PayMode: mode, OutTradeNo: "order-1", AmountCents: 1299, ExpireAt: time.Now().Add(time.Hour), OpenID: "openid-1", PayerClientIP: "192.0.2.1"}
		created, err := channel.CreateOrder(context.Background(), request)
		if err != nil {
			t.Fatalf("CreateOrder(%s): %v", mode, err)
		}
		switch mode {
		case PayModeNative:
			if created.CodeURL != "weixin://qr" {
				t.Fatalf("Native result = %+v", created)
			}
		case PayModeJSAPI:
			if created.JSAPIParams == nil || created.JSAPIParams.PaySign != "signature" {
				t.Fatalf("JSAPI result = %+v", created)
			}
		case PayModeH5:
			if created.H5URL != "https://pay.invalid/h5" {
				t.Fatalf("H5 result = %+v", created)
			}
		}
	}
	queried, err := channel.QueryOrder(context.Background(), "order-1")
	if err != nil || queried.Status != "SUCCESS" || queried.AmountCents != 1299 {
		t.Fatalf("QueryOrder() = (%+v, %v)", queried, err)
	}
	refunded, err := channel.Refund(context.Background(), &RefundRequest{OutTradeNo: "order-1", OutRefundNo: "refund-1", RefundAmountCents: 300, TotalAmountCents: 1299})
	if err != nil || refunded.Status != "SUCCESS" || refunded.RefundAmountCents != 300 {
		t.Fatalf("Refund() = (%+v, %v)", refunded, err)
	}
	refundQuery, err := channel.QueryRefund(context.Background(), "order-1", "refund-1")
	if err != nil || refundQuery.Status != "SUCCESS" || refundQuery.RefundAmountCents != 300 {
		t.Fatalf("QueryRefund() = (%+v, %v)", refundQuery, err)
	}
	if fake.nativeCalls != 1 || fake.jsapiCalls != 1 || fake.h5Calls != 1 || fake.queryCalls != 1 || fake.refundCalls != 1 || fake.refundQueryCalls != 1 || fake.paySignCalls != 1 {
		t.Fatalf("unexpected SDK call counts: %+v", fake)
	}
}

func TestChannelFailureAcknowledgements(t *testing.T) {
	alipayAck := (&AlipayChannel{}).FailureAck(errors.New("invalid callback"))
	if alipayAck.AckStatus != http.StatusOK || alipayAck.AckBody != "fail" || alipayAck.AckContentType != "text/plain; charset=utf-8" {
		t.Fatalf("unexpected alipay failure acknowledgement: %+v", alipayAck)
	}
	wechatAck := (&WechatChannel{}).FailureAck(errors.New("invalid callback"))
	if wechatAck.AckStatus != http.StatusInternalServerError || wechatAck.AckContentType != "application/json; charset=utf-8" {
		t.Fatalf("unexpected wechat failure acknowledgement: %+v", wechatAck)
	}
	var body map[string]string
	if err := json.Unmarshal([]byte(wechatAck.AckBody), &body); err != nil || body["code"] != "FAIL" {
		t.Fatalf("wechat failure acknowledgement body = %q, err = %v", wechatAck.AckBody, err)
	}
}

func TestAlipayCallbackRejectsInvalidSignature(t *testing.T) {
	form := url.Values{
		"trade_status": {"TRADE_SUCCESS"},
		"out_trade_no": {"order-1"},
		"total_amount": {"1.00"},
		"sign":         {"invalid"},
		"sign_type":    {"RSA2"},
	}
	request, err := http.NewRequest(http.MethodPost, "/callback", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	channel := &AlipayChannel{config: AlipayConfig{PublicKey: "invalid-public-key"}}
	if _, err := channel.HandleCallback(context.Background(), request, CallbackMeta{}); err == nil {
		t.Fatal("HandleCallback accepted an invalid Alipay signature")
	}
}

func TestAlipayCallbackVerifiesSuccessfulPayment(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	publicKeyDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	form := make(gopay.BodyMap)
	form.Set("trade_status", "TRADE_SUCCESS").
		Set("out_trade_no", "order-1").
		Set("trade_no", "ali-trade-1").
		Set("total_amount", "12.99").
		Set("gmt_payment", "2026-10-10 12:00:00")
	signature, err := alipaySDKPkg.GetRsaSign(form, alipaySDKPkg.RSA2, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	form.Set("sign_type", "RSA2").Set("sign", signature)
	request, err := http.NewRequest(http.MethodPost, "/callback", strings.NewReader(form.EncodeURLParams()))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	channel := &AlipayChannel{config: AlipayConfig{PublicKey: base64.StdEncoding.EncodeToString(publicKeyDER)}}
	result, err := channel.HandleCallback(context.Background(), request, CallbackMeta{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "SUCCESS" || result.AmountCents != 1299 || result.OutTradeNo != "order-1" || result.AckBody != "success" {
		t.Fatalf("unexpected successful callback result: %+v", result)
	}
}

func TestWechatCallbackRejectsInvalidSignature(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: mustMarshalPKCS8(t, privateKey)})
	client, err := wechat.NewClientV3("mch", "serial", "0123456789abcdef0123456789abcdef", string(privateKeyPEM))
	if err != nil {
		t.Fatal(err)
	}
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: mustMarshalPKIX(t, &privateKey.PublicKey)})
	if err := client.AutoVerifySignByPublicKey(publicKeyPEM, "platform-key-id"); err != nil {
		t.Fatal(err)
	}
	channel := &WechatChannel{client: client}
	body := `{"id":"notify-1","create_time":"2025-01-01T00:00:00+08:00","event_type":"TRANSACTION.SUCCESS","resource_type":"encrypt-resource","resource":{"algorithm":"AEAD_AES_256_GCM","ciphertext":"invalid","nonce":"123456789012","associated_data":""}}`
	request, err := http.NewRequest(http.MethodPost, "/callback", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Wechatpay-Timestamp", "1735660800")
	request.Header.Set("Wechatpay-Nonce", "nonce")
	request.Header.Set("Wechatpay-Signature", "invalid")
	request.Header.Set("Wechatpay-Serial", "platform-key-id")
	if _, err := channel.HandleCallback(context.Background(), request, CallbackMeta{}); err == nil {
		t.Fatal("HandleCallback accepted an invalid WeChat signature")
	}
}

func TestWechatCallbackVerifiesAndDecryptsSuccessfulPayment(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: mustMarshalPKCS8(t, privateKey)})
	const apiV3Key = "0123456789abcdef0123456789abcdef"
	client, err := wechat.NewClientV3("mch", "serial", apiV3Key, string(privateKeyPEM))
	if err != nil {
		t.Fatal(err)
	}
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: mustMarshalPKIX(t, &privateKey.PublicKey)})
	if err := client.AutoVerifySignByPublicKey(publicKeyPEM, "platform-key-id"); err != nil {
		t.Fatal(err)
	}

	const nonce = "123456789012"
	const associatedData = "payment-notify"
	resource, err := encryptNotifyResource([]byte(apiV3Key), nonce, associatedData,
		`{"appid":"app","mchid":"mch","out_trade_no":"order-1","transaction_id":"tx-1","trade_state":"SUCCESS","success_time":"2026-10-10T12:00:00+08:00","amount":{"total":1299}}`)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"id": "notify-1", "create_time": "2026-10-10T12:00:00+08:00",
		"event_type": "TRANSACTION.SUCCESS", "resource_type": "encrypt-resource",
		"resource": map[string]string{"algorithm": "AEAD_AES_256_GCM", "ciphertext": base64.StdEncoding.EncodeToString(resource), "nonce": nonce, "associated_data": associatedData},
	})
	if err != nil {
		t.Fatal(err)
	}
	const timestamp = "1791633600"
	const headerNonce = "header-nonce"
	signed := []byte(timestamp + "\n" + headerNonce + "\n" + string(payload) + "\n")
	digest := sha256.Sum256(signed)
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPost, "/callback", strings.NewReader(string(payload)))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Wechatpay-Timestamp", timestamp)
	request.Header.Set("Wechatpay-Nonce", headerNonce)
	request.Header.Set("Wechatpay-Signature", base64.StdEncoding.EncodeToString(signature))
	request.Header.Set("Wechatpay-Serial", "platform-key-id")

	result, err := (&WechatChannel{client: client}).HandleCallback(context.Background(), request, CallbackMeta{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "SUCCESS" || result.OutTradeNo != "order-1" || result.AmountCents != 1299 || result.AckBody != `{"code":"SUCCESS","message":"成功"}` {
		t.Fatalf("unexpected successful callback result: %+v", result)
	}
}

func encryptNotifyResource(key []byte, nonce, associatedData, plaintext string) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Seal(nil, []byte(nonce), []byte(plaintext), []byte(associatedData)), nil
}

func mustMarshalPKCS8(t *testing.T, key *rsa.PrivateKey) []byte {
	t.Helper()
	encoded, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func mustMarshalPKIX(t *testing.T, key *rsa.PublicKey) []byte {
	t.Helper()
	encoded, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

type testChannel struct {
	name  string
	modes []string
}

type fakeAlipaySDK struct {
	createCalls, queryCalls, refundCalls, refundQueryCalls int
}

func (fake *fakeAlipaySDK) TradePrecreate(context.Context, gopay.BodyMap) (*alipaySDKPkg.TradePrecreateResponse, error) {
	fake.createCalls++
	return &alipaySDKPkg.TradePrecreateResponse{Response: &alipaySDKPkg.TradePrecreate{QrCode: "alipay://qr"}}, nil
}
func (fake *fakeAlipaySDK) TradeQuery(context.Context, gopay.BodyMap) (*alipaySDKPkg.TradeQueryResponse, error) {
	fake.queryCalls++
	return &alipaySDKPkg.TradeQueryResponse{Response: &alipaySDKPkg.TradeQuery{OutTradeNo: "order-1", TradeStatus: "TRADE_SUCCESS", TotalAmount: "12.99"}}, nil
}
func (fake *fakeAlipaySDK) TradeRefund(context.Context, gopay.BodyMap) (*alipaySDKPkg.TradeRefundResponse, error) {
	fake.refundCalls++
	return &alipaySDKPkg.TradeRefundResponse{Response: &alipaySDKPkg.TradeRefund{OutTradeNo: "order-1", FundChange: "Y", GmtRefundPay: "2026-10-10 12:00:00"}}, nil
}
func (fake *fakeAlipaySDK) TradeFastPayRefundQuery(context.Context, gopay.BodyMap) (*alipaySDKPkg.TradeFastpayRefundQueryResponse, error) {
	fake.refundQueryCalls++
	return &alipaySDKPkg.TradeFastpayRefundQueryResponse{Response: &alipaySDKPkg.TradeRefundQuery{OutTradeNo: "order-1", OutRequestNo: "refund-1", RefundStatus: "REFUND_SUCCESS", RefundAmount: "3.00"}}, nil
}

type fakeWechatSDK struct {
	nativeCalls, jsapiCalls, h5Calls, queryCalls, refundCalls, refundQueryCalls, paySignCalls int
}

func (fake *fakeWechatSDK) V3TransactionNative(context.Context, gopay.BodyMap) (*wechat.NativeRsp, error) {
	fake.nativeCalls++
	return &wechat.NativeRsp{Code: wechat.Success, Response: &wechat.Native{CodeUrl: "weixin://qr"}}, nil
}
func (fake *fakeWechatSDK) V3TransactionJsapi(context.Context, gopay.BodyMap) (*wechat.PrepayRsp, error) {
	fake.jsapiCalls++
	return &wechat.PrepayRsp{Code: wechat.Success, Response: &wechat.Prepay{PrepayId: "prepay-1"}}, nil
}
func (fake *fakeWechatSDK) V3TransactionH5(context.Context, gopay.BodyMap) (*wechat.H5Rsp, error) {
	fake.h5Calls++
	return &wechat.H5Rsp{Code: wechat.Success, Response: &wechat.H5Url{H5Url: "https://pay.invalid/h5"}}, nil
}
func (fake *fakeWechatSDK) V3TransactionQueryOrder(context.Context, wechat.OrderNoType, string) (*wechat.QueryOrderRsp, error) {
	fake.queryCalls++
	return &wechat.QueryOrderRsp{Code: wechat.Success, Response: &wechat.QueryOrder{OutTradeNo: "order-1", TradeState: "SUCCESS", Amount: &wechat.Amount{Total: 1299}}}, nil
}
func (fake *fakeWechatSDK) V3Refund(context.Context, gopay.BodyMap) (*wechat.RefundRsp, error) {
	fake.refundCalls++
	return &wechat.RefundRsp{Code: wechat.Success, Response: &wechat.RefundOrderResponse{OutTradeNo: "order-1", OutRefundNo: "refund-1", RefundId: "wx-refund-1", Status: "SUCCESS", Amount: &wechat.RefundOrderAmount{Refund: 300, Total: 1299}}}, nil
}
func (fake *fakeWechatSDK) V3RefundQuery(context.Context, string, gopay.BodyMap) (*wechat.RefundQueryRsp, error) {
	fake.refundQueryCalls++
	return &wechat.RefundQueryRsp{Code: wechat.Success, Response: &wechat.RefundQueryResponse{OutTradeNo: "order-1", OutRefundNo: "refund-1", RefundId: "wx-refund-1", Status: "SUCCESS", Amount: &wechat.RefundOrderAmount{Refund: 300, Total: 1299}}}, nil
}
func (fake *fakeWechatSDK) PaySignOfJSAPI(string, string) (*wechat.JSAPIPayParams, error) {
	fake.paySignCalls++
	return &wechat.JSAPIPayParams{AppId: "app-1", PaySign: "signature"}, nil
}

func (ch *testChannel) Name() string       { return ch.name }
func (ch *testChannel) PayModes() []string { return ch.modes }
func (ch *testChannel) CreateOrder(context.Context, *CreateOrderRequest) (*CreateOrderResult, error) {
	return nil, nil
}
func (ch *testChannel) QueryOrder(context.Context, string) (*OrderQueryResult, error) {
	return nil, nil
}
func (ch *testChannel) Refund(context.Context, *RefundRequest) (*RefundResult, error) {
	return nil, nil
}
func (ch *testChannel) QueryRefund(context.Context, string, string) (*RefundResult, error) {
	return nil, nil
}
func (ch *testChannel) HandleCallback(context.Context, *http.Request, CallbackMeta) (*CallbackResult, error) {
	return nil, nil
}
func (ch *testChannel) FailureAck(error) *CallbackResult { return nil }
