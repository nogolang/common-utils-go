package smsUtils

// 客户端形态的短信发送（抽自参考项目 xianyu-open 的 sms.Client）。
//
// 与同包 `Send(ctx, cfg, phone, templateCode, params)` 的区别：
//   - `Send` 是**函数式**：每次调用都按传入的模板码与变量发一条，适合"验证码 / 业务提醒各发各的"；
//   - 这里的 `Client` 是**对象式**：构造时固定"账号 + 签名 + 一个模板码"，之后 `SendCode(phone, code)`
//     一行发一条，**并且返回 SendResult（含阿里云回执号 BizId / RequestId）**。
//
// 什么时候用 Client：用户报"收不到验证码"时，BizId 能直接在阿里云后台查到这条短信的投递明细
// （被拦截/黑名单/模板不符都有迹可循）——`Send` 只返回一个 error，定位不到具体那一条。
// 常规业务链路继续用 `Send` 即可，不必都上 Client。
import (
	"context"
	"fmt"
	"time"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi "github.com/alibabacloud-go/dysmsapi-20170525/v5/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/pkg/errors"
)

// ClientConfig 短信客户端配置（Endpoint 留空 = 官方地址）
type ClientConfig struct {
	AccessKeyId     string
	AccessKeySecret string
	SignName        string
	TemplateCode    string
	Endpoint        string
}

// SendResult 发送结果（**含阿里云回执号**，排障靠它）
type SendResult struct {
	// Success 业务是否成功（阿里云返回码 == OK）
	Success bool
	// Code 阿里云返回码，OK 表示成功
	Code string
	// Message 阿里云返回消息
	Message string
	// BizId 发送回执 ID：在阿里云控制台/日志服务里凭它查这条短信的投递明细
	BizId string
	// RequestId 阿里云请求 ID
	RequestId string
}

// Client 短信客户端（固定账号 + 签名 + 一个模板码）
type Client struct {
	client       *dysmsapi.Client
	signName     string
	templateCode string
}

// NewClient 创建短信客户端
func NewClient(cfg *ClientConfig) (*Client, error) {
	if cfg == nil {
		return nil, errors.New("短信客户端配置为空")
	}
	if err := (Config{
		AccessKeyId:     cfg.AccessKeyId,
		AccessKeySecret: cfg.AccessKeySecret,
		SignName:        cfg.SignName,
	}).Validate(); err != nil {
		return nil, err
	}
	if cfg.TemplateCode == "" {
		return nil, errors.New("短信配置缺失（TemplateCode）")
	}
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	client, err := dysmsapi.NewClient(&openapi.Config{
		AccessKeyId:     tea.String(cfg.AccessKeyId),
		AccessKeySecret: tea.String(cfg.AccessKeySecret),
		Endpoint:        tea.String(endpoint),
	})
	if err != nil {
		return nil, errors.Wrap(err, "创建短信客户端失败")
	}
	return &Client{client: client, signName: cfg.SignName, templateCode: cfg.TemplateCode}, nil
}

// SendCode 用构造时固定的模板发一条验证码短信（模板变量 {"code":"xxxxxx"}）
//
// 返回值语义（与参考项目一致，调用方必须看 Success，**不能只看 err**）：
// 传输/参数类问题 err != nil；阿里云业务拒绝（签名不匹配、模板不存在、手机号黑名单）时
// err == nil 但 Success == false，Message 里有原因。
func (c *Client) SendCode(ctx context.Context, phone, code string) (*SendResult, error) {
	if c == nil || c.client == nil {
		return nil, errors.New("短信客户端未初始化")
	}
	if phone == "" {
		return nil, errors.New("手机号为空")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, dialTimeout)
	defer cancel()

	type result struct {
		res *SendResult
		err error
	}
	done := make(chan result, 1)
	go func() {
		resp, err := c.client.SendSms(&dysmsapi.SendSmsRequest{
			PhoneNumbers:  tea.String(phone),
			SignName:      tea.String(c.signName),
			TemplateCode:  tea.String(c.templateCode),
			TemplateParam: tea.String(fmt.Sprintf(`{"code":"%s"}`, code)),
		})
		if err != nil {
			done <- result{err: errors.Wrap(err, "短信 API 调用失败")}
			return
		}
		if resp == nil || resp.Body == nil {
			done <- result{err: errors.New("短信发送失败：响应为空")}
			return
		}
		res := &SendResult{
			Code:      tea.StringValue(resp.Body.Code),
			Message:   tea.StringValue(resp.Body.Message),
			BizId:     tea.StringValue(resp.Body.BizId),
			RequestId: tea.StringValue(resp.Body.RequestId),
		}
		res.Success = res.Code == "OK"
		done <- result{res: res}
	}()

	select {
	case <-ctx.Done():
		return nil, errors.New("短信发送超时或被取消")
	case r := <-done:
		return r.res, r.err
	}
}

// SendWithTimeoutOf 客户端形态的带默认超时发送（不传 ctx 时用）
func (c *Client) SendWithTimeoutOf(phone, code string) (*SendResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return c.SendCode(ctx, phone, code)
}
