package smsUtils

// 阿里云验证码 2.0（滑块 / 无痕）客户端 —— 抽自参考项目 xianyu-open-merchant/internal/handle/captcha。
//
// 用途：发短信/邮件这类"有成本"的通道前，先让前端过一道人机校验（前端拿 SDK 生成
// captchaVerifyParam，后端拿它向阿里云验证），挡住脚本批量刷码。
//
// 「无痕」（TRACELESS）**后端不感知**——那是阿里云控制台里场景的配置，后端只透传 captchaVerifyParam。
// 错误码语义同样不在本包 switch（T001 通过 / F007 场景非法 / F008 重复提交 / F017 疑似攻击），
// 只通过 VerifyResult.VerifyCode 透传给业务侧，由业务侧决定怎么提示。
//
// 相比参考项目补了两处（两边原实现都有）：
//  ① **ctx 超时**：阿里云 SDK 无内置超时，Verify 不接 ctx 时接口卡住会把 goroutine 挂死
//     （在 HTTP 请求路径上就是接口悬挂），与本包 sms Send 的 10s 口径一致；
//  ② **nil Body 检查**：SDK 异常返回时 resp.Body 为 nil，直接解引用会 panic。
import (
	"context"
	"time"

	captcha "github.com/alibabacloud-go/captcha-20230305/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/pkg/errors"
)

// captchaEndpoint 阿里云验证码服务地址（国内站）
const captchaEndpoint = "captcha.aliyuncs.com"

// captchaTimeout 单次校验超时（与本包 Send 同口径）
const captchaTimeout = 5 * time.Second

// CaptchaClientConfig 验证码 2.0 客户端配置（全部由调用方注入，本包不读配置文件）
type CaptchaClientConfig struct {
	// AccessKeyId / AccessKeySecret 阿里云凭据（与短信通常同一主账号）
	AccessKeyId     string
	AccessKeySecret string
	// SceneId 场景 id（前端 SDK 与后端必须一致；空则每次调用都要显式传）
	SceneId string
	// Endpoint 可选，留空 = 官方地址
	Endpoint string
}

// CaptchaVerifyResult 校验结果
type CaptchaVerifyResult struct {
	// Success 是否通过（= CaptchaPass）
	Success bool
	// CaptchaPass 阿里云侧的校验结论
	CaptchaPass bool
	// VerifyCode 结果码（T001=通过, F007=场景不合法, F008=重复提交, F017=疑似攻击）
	VerifyCode string
	// RequestId 阿里云请求 id（排障用：可凭它找阿里云查这次请求）
	RequestId string
}

// CaptchaClient 阿里云验证码 2.0 客户端
type CaptchaClient struct {
	client  *captcha.Client
	sceneId string
}

// NewCaptchaClient 创建验证码 2.0 客户端（Endpoint 留空用官方地址）
func NewCaptchaClient(cfg CaptchaClientConfig) (*CaptchaClient, error) {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = captchaEndpoint
	}
	client, err := captcha.NewClient(&openapi.Config{
		AccessKeyId:     tea.String(cfg.AccessKeyId),
		AccessKeySecret: tea.String(cfg.AccessKeySecret),
		Endpoint:        tea.String(endpoint),
	})
	if err != nil {
		return nil, errors.Wrap(err, "创建验证码2.0客户端失败")
	}
	return &CaptchaClient{client: client, sceneId: cfg.SceneId}, nil
}

// Verify 校验前端传来的 captchaVerifyParam；sceneId 传空则用构造时的默认场景
//
// 注意：**不要在本包里 switch VerifyCode 做业务判断**——不同业务对 F017（疑似攻击）的处置不同
// （登录接口可能直接封、绑定接口可能只记日志），由调用方按 VerifyCode 自己决定。
func (c *CaptchaClient) Verify(ctx context.Context, captchaVerifyParam, sceneId string) (*CaptchaVerifyResult, error) {
	if captchaVerifyParam == "" {
		return nil, errors.New("captchaVerifyParam 为空")
	}
	if sceneId == "" {
		sceneId = c.sceneId
	}
	if sceneId == "" {
		return nil, errors.New("验证码场景 id 未配置（构造时或调用时二选一）")
	}
	if c.client == nil {
		return nil, errors.New("验证码客户端未初始化")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, captchaTimeout)
	defer cancel()

	type result struct {
		res *CaptchaVerifyResult
		err error
	}
	done := make(chan result, 1)
	go func() {
		resp, err := c.client.VerifyIntelligentCaptcha(&captcha.VerifyIntelligentCaptchaRequest{
			CaptchaVerifyParam: tea.String(captchaVerifyParam),
			SceneId:            tea.String(sceneId),
		})
		if err != nil {
			done <- result{err: errors.Wrap(err, "验证码2.0 API 调用失败")}
			return
		}
		if resp == nil || resp.Body == nil {
			done <- result{err: errors.New("验证码2.0 响应为空")}
			return
		}
		res := &CaptchaVerifyResult{RequestId: tea.StringValue(resp.Body.RequestId)}
		if r := resp.Body.Result; r != nil {
			res.VerifyCode = tea.StringValue(r.VerifyCode)
			res.CaptchaPass = tea.BoolValue(r.VerifyResult)
		}
		res.Success = res.CaptchaPass
		done <- result{res: res}
	}()

	select {
	case <-ctx.Done():
		return nil, errors.New("验证码2.0 校验超时或被取消")
	case r := <-done:
		return r.res, r.err
	}
}
