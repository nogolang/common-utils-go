# payUtils

`payUtils` contains the reusable Alipay and WeChat payment protocol adapters. It handles channel requests, platform response normalization, callback signature verification, callback decryption, and provider acknowledgement formatting. Applications keep their order ledger, persistence, routes, and business state transitions.

Construct one adapter per merchant configuration:

```go
alipayChannel, err := payUtils.NewAlipayChannel(payUtils.AlipayConfig{
	AppID:         appID,
	AppPrivateKey: decryptedPrivateKey,
	PublicKey:     alipayPublicKey,
	NotifyURL:     "https://example.com/pay/alipay/callback",
	IsProduction:  true,
})
if err != nil {
	return err
}

wechatChannel, err := payUtils.NewWechatChannel(payUtils.WechatConfig{
	AppID:           appID,
	MchID:           mchID,
	SerialNo:        merchantCertSerial,
	PrivateKey:      decryptedMerchantPrivateKey,
	APIV3Key:        decryptedAPIV3Key,
	MchCertPEM:      merchantCertPEM,
	PayPublicKeyID:  wechatPublicKeyID,
	PayPublicKeyPEM: wechatPublicKeyPEM,
	NativeNotifyURL: "https://example.com/pay/wechat/native/callback",
	JSAPINotifyURL:  "https://example.com/pay/wechat/js/callback",
	H5NotifyURL:     "https://example.com/pay/wechat/h5/callback",
})
if err != nil {
	return err
}
```

The application supplies plaintext key material after retrieving its decryption key from the deployment environment or a secret manager. The package does not load application configuration or decrypt stored credentials.

All amounts passed through the interface are integer cents. `CreateOrderRequest.PayMode` accepts `PayModeQRCode`, `PayModeNative`, `PayModeJSAPI`, or `PayModeH5`, depending on the adapter. Callback `Status` is cross-channel normalized; applications map it to their own ledger states.
