// Package payUtils provides reusable adapters for online payment channels.
// It owns platform protocol details; applications own order persistence and
// business state transitions.
package payUtils

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	Alipay = "alipay"
	Wechat = "wechat"

	PayModeQRCode = "QR_CODE_OFFLINE"
	PayModeNative = "NATIVE"
	PayModeJSAPI  = "JSAPI"
	PayModeH5     = "H5"
)

var (
	ErrUnknownChannel     = errors.New("unknown payment channel")
	ErrPayModeUnsupported = errors.New("payment mode is not supported")
)

type ProviderError struct {
	Message string
	Data    any
}

func (err *ProviderError) Error() string { return err.Message }

type AlipayConfig struct {
	AppID         string
	AppPrivateKey string
	PublicKey     string
	NotifyURL     string
	IsProduction  bool
}

type WechatConfig struct {
	AppID           string
	MchID           string
	SerialNo        string
	PrivateKey      string
	APIV3Key        string
	MchCertPEM      string
	PayPublicKeyID  string
	PayPublicKeyPEM string
	NativeNotifyURL string
	JSAPINotifyURL  string
	H5NotifyURL     string
}

type CreateOrderRequest struct {
	PayMode       string
	OutTradeNo    string
	Subject       string
	AmountCents   int64
	ExpireSeconds int
	ExpireAt      time.Time
	OpenID        string
	PayerClientIP string
}

type RefundRequest struct {
	OutTradeNo        string
	OutRefundNo       string
	RefundAmountCents int64
	TotalAmountCents  int64
	Reason            string
}

type CallbackMeta struct{ RealIP string }

type JSAPIParams struct {
	AppID     string `json:"appId"`
	TimeStamp string `json:"timeStamp"`
	NonceStr  string `json:"nonceStr"`
	Package   string `json:"package"`
	SignType  string `json:"signType"`
	PaySign   string `json:"paySign"`
}

type CreateOrderResult struct {
	Channel           string
	PayMode           string
	OutTradeNo        string
	ExpireSeconds     int
	CodeURL           string
	H5URL             string
	JSAPIParams       *JSAPIParams
	PayPayload        string
	PendingTradeState string
	Raw               any
}

type OrderQueryResult struct {
	Channel            string     `json:"channel"`
	OutTradeNo         string     `json:"outTradeNo"`
	Status             string     `json:"status"`
	PlatformTradeState string     `json:"platformTradeState"`
	TransactionID      string     `json:"transactionId,omitempty"`
	AmountCents        int64      `json:"amountCents"`
	PaidAt             *time.Time `json:"paidAt,omitempty"`
	Raw                any        `json:"raw"`
}

type RefundResult struct {
	Channel           string     `json:"channel"`
	OutTradeNo        string     `json:"outTradeNo"`
	OutRefundNo       string     `json:"outRefundNo"`
	RefundID          string     `json:"refundId,omitempty"`
	Status            string     `json:"status"`
	PlatformState     string     `json:"platformState,omitempty"`
	RefundAmountCents int64      `json:"refundAmountCents"`
	TotalAmountCents  int64      `json:"totalAmountCents"`
	SuccessAt         *time.Time `json:"successAt,omitempty"`
	Raw               any        `json:"raw"`
}

type CallbackResult struct {
	Channel        string
	OutTradeNo     string
	TransactionID  string
	TradeState     string
	Status         string
	AmountCents    int64
	PaidAt         *time.Time
	AckStatus      int
	AckContentType string
	AckBody        string
}

type PayChannel interface {
	Name() string
	PayModes() []string
	CreateOrder(context.Context, *CreateOrderRequest) (*CreateOrderResult, error)
	QueryOrder(context.Context, string) (*OrderQueryResult, error)
	Refund(context.Context, *RefundRequest) (*RefundResult, error)
	QueryRefund(context.Context, string, string) (*RefundResult, error)
	HandleCallback(context.Context, *http.Request, CallbackMeta) (*CallbackResult, error)
	FailureAck(error) *CallbackResult
}

type Registry struct{ channels map[string]PayChannel }

func NewRegistry(channels ...PayChannel) *Registry {
	registry := &Registry{channels: make(map[string]PayChannel, len(channels))}
	for _, item := range channels {
		if item != nil {
			registry.channels[strings.ToLower(item.Name())] = item
		}
	}
	return registry
}

func (registry *Registry) Get(name string) (PayChannel, error) {
	item, ok := registry.channels[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return nil, fmt.Errorf("unsupported payment channel %q: %w", name, ErrUnknownChannel)
	}
	return item, nil
}

func (registry *Registry) Names() []string {
	names := make([]string, 0, len(registry.channels))
	for name := range registry.channels {
		names = append(names, name)
	}
	return names
}

func Supports(ch PayChannel, payMode string) bool {
	for _, item := range ch.PayModes() {
		if item == payMode {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
