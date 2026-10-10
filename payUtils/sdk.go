package payUtils

import (
	"context"

	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/alipay"
	wechatv3 "github.com/go-pay/gopay/wechat/v3"
)

type alipaySDK interface {
	TradePrecreate(context.Context, gopay.BodyMap) (*alipay.TradePrecreateResponse, error)
	TradeQuery(context.Context, gopay.BodyMap) (*alipay.TradeQueryResponse, error)
	TradeRefund(context.Context, gopay.BodyMap) (*alipay.TradeRefundResponse, error)
	TradeFastPayRefundQuery(context.Context, gopay.BodyMap) (*alipay.TradeFastpayRefundQueryResponse, error)
}

type wechatSDK interface {
	V3TransactionNative(context.Context, gopay.BodyMap) (*wechatv3.NativeRsp, error)
	V3TransactionJsapi(context.Context, gopay.BodyMap) (*wechatv3.PrepayRsp, error)
	V3TransactionH5(context.Context, gopay.BodyMap) (*wechatv3.H5Rsp, error)
	V3TransactionQueryOrder(context.Context, wechatv3.OrderNoType, string) (*wechatv3.QueryOrderRsp, error)
	V3Refund(context.Context, gopay.BodyMap) (*wechatv3.RefundRsp, error)
	V3RefundQuery(context.Context, string, gopay.BodyMap) (*wechatv3.RefundQueryRsp, error)
	PaySignOfJSAPI(string, string) (*wechatv3.JSAPIPayParams, error)
}
