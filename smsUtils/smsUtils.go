// Package smsUtils 短信发送（阿里云 dysmsapi v5）
//
// 2026-09-26 从 shop-common/commonUtil/smsUtils 迁入公共库；
// 2026-09-27 加固：格式校验前置 + 阿里云错误码可读化 + 客户端复用 + 结构化结果。
//
// 配置口径：**只认传进来的结构体，不读任何配置文件**（与 gormUtils / redisUtils 同款）。
// 凭据与签名/模板码的**来源由调用方决定**——本项目（3-my-shop-single）的口径是：
//   - **AccessKeyId / AccessKeySecret 复用共享配置 `common.*.yaml` 的 `aliYunAccount` 段**
//     （与 OSS 上传、内容审核**同一阿里云账号**，由各服务启动时装配成本包的 Config）；
//   - SignName / TemplateCode / RemindTemplateCode 来自业务配置 sys_business_config
//     （运营可在管理端「业务配置」页自助修改，改完即时生效、**不需要重启**）；
//   - 本包一概不关心来源，因此不依赖 viper。
//
// 本包一概不关心来源，因此不依赖 viper（迁入前它有两处 viper 直读，已删除）。
//
// 四条纪律：
//  1. **必须带超时**：阿里云 SDK 无内置超时，卡住会把调用方 goroutine 挂死
//     （在 HTTP 请求路径上就是接口悬挂）。ctx 层 10s + SDK 层 Connect/Read 超时双保险。
//  2. **调用方必须容错**：发送失败绝不能让业务主流程失败——通知类短信失败只记日志
//     （见 shop-server-admin 的 noticeSendHandler 注释：返回 error 会触发 MQ 重试 → 重复发）。
//  3. **模板码由调用方显式传**：验证码模板与业务提醒模板是不同模板、变量也不同
//     （见 Send 的 templateCode 参数），不要用 Config 里的字段替调用方做选择。
//  4. **配置不全就别发**：格式/必填校验不过时**直接返回可读错误、不发请求**
//     （2026-09-27 实测教训：AK 已被阿里云停用时，旧实现"只判非空"照样显示"就绪"并发请求，
//     拿到的却是 `InvalidAccessKeyId.NotFound` 这类天书错误码）。
//     ⚠️ 本包**不引入**"配置不全就跳过真实发送、验证码照常写库"的测试模式——那是安全隐患
//     （验证码不发却能登录）；配置不全一律明确报错。
package smsUtils

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi "github.com/alibabacloud-go/dysmsapi-20170525/v5/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/pkg/errors"
)

// defaultEndpoint 阿里云短信服务地址（国内站；海外站另议）
const defaultEndpoint = "dysmsapi.aliyuncs.com"

// dialTimeout 单次调用超时（ctx 层；与 mailUtils 同口径）
const dialTimeout = 10 * time.Second

// sdkTimeout SDK 自身的连接/读取超时（2026-09-27 补：原来只有 ctx 兜底，
// SDK 内部 HTTP 客户端无超时，网关半开时会一直挂着直到 ctx 生效）
const sdkTimeout = 8 * time.Second

// 格式校验用（2026-09-27 加严，替代原"只判非空"）
var (
	// 阿里云 AccessKeyId 固定 LTAI 开头（16~24 位）；Secret 长度 16~30
	akIDPattern   = regexp.MustCompile(`^LTAI[A-Za-z0-9]{12,}$`)
	akSecPattern  = regexp.MustCompile(`^[A-Za-z0-9+/=]{16,}$`)
	phonePattern  = regexp.MustCompile(`^1[3-9]\d{9}$`)
	aliyunCodePat = regexp.MustCompile(`^SMS_[A-Za-z0-9]+$`)
)

// Config 短信发送所需配置，**全部由调用方注入**
type Config struct {
	// AccessKeyId / AccessKeySecret 阿里云账号凭据。
	// 与 OSS 上传、内容审核通常共用同一主账号，但**从哪取由调用方决定**
	// （本项目从共享配置的 CommonConfig.AliYunAccount 拿，保证与 OSS 同一账号；
	//  ⚠️ 改它要改 common.*.yaml 并重启服务 —— 业务配置页里改不到）。
	AccessKeyId     string
	AccessKeySecret string
	// SignName 短信签名（如「深圳市龙岗区深空软件」；必须与 AccessKey 同属一个阿里云账号）
	SignName string
	// TemplateCode 验证码模板码（变量 code）；**留空 = 发不出短信验证码**
	TemplateCode string
	// RemindTemplateCode 业务提醒模板码（变量 days/autoDays/orderNo）；**留空 = 不发业务短信**
	RemindTemplateCode string
	// CaptchaSceneId 验证码 2.0 场景 id（滑块/智能验证用，暂未接入）
	CaptchaSceneId string
}

// RemindTemplate 业务提醒短信模板码（值类型包装）
//
// 为什么不注入裸 string：依赖注入图里"哪儿都能满足一个 string"，注入点不显式；
// 且类型上就把「业务提醒模板」与「验证码模板」分开，不会哪天把验证码模板码传进
// 确认收货提醒（两者变量名对不上，阿里云直接判失败）。
type RemindTemplate struct {
	Code string
}

// Fingerprint 配置指纹（打日志用；**只露 AK 后 4 位**，绝不输出完整 AK/Secret）
//
// 2026-09-27 加：排查"短信发不出去"时第一件事是确认**读到的到底是哪份配置**，
// 而完整 AK 打日志是安全事故（截图/日志外泄即账号泄露）。
// 本项目 AK 复用 common.*.yaml 的 aliYunAccount（与 OSS 同账号），
// 所以"短信的 AK"应与 OSS 日志里出现的后 4 位一致——不一致就说明装配链断了。
func (c Config) Fingerprint() string {
	return fmt.Sprintf("ak=%s sign=%q captchaTpl=%q remindTpl=%q",
		MaskSecret(c.AccessKeyId), c.SignName, c.TemplateCode, c.RemindTemplateCode)
}

// Validate 必填 + **格式**校验（2026-09-27 加严）
//
// 为什么不只判非空：AK 已被停用/删除时，非空校验照样放行 → 真发一次请求 →
// 拿回 `InvalidAccessKeyId.NotFound`（"指定 access key 不存在"）这类天书，
// 而日志里"短信AK就绪: true"让人以为配置没问题。格式层面能挡掉的（填错段、贴了密钥、
// 带了引号/空格、复制了 accessKeyId 当 secret）**在这一步就该拦下，不浪费一次请求**。
// 至于 AK **是否真实存在且有短信权限**只能问阿里云（代码判不了），但那时错误已被翻译成中文。
func (c Config) Validate() error {
	akID := strings.TrimSpace(c.AccessKeyId)
	akSecret := strings.TrimSpace(c.AccessKeySecret)
	if akID == "" || akSecret == "" {
		return errors.New("短信配置缺失（AccessKeyId / AccessKeySecret）：" +
			"请在 common.*.yaml 的 aliYunAccount 段配置（与 OSS 共用同一阿里云账号，改后需重启服务）")
	}
	if !akIDPattern.MatchString(akID) {
		return errors.Errorf("短信配置 AccessKeyId 格式异常（应以 LTAI 开头）：当前值 %s。"+
			"请检查 common.*.yaml 的 aliYunAccount.accessKeyId（注意别把 Secret 填到这一行）", MaskSecret(akID))
	}
	if !akSecPattern.MatchString(akSecret) {
		return errors.New("短信配置 AccessKeySecret 格式异常（长度不足或含非 base64 字符）：" +
			"请检查 common.*.yaml 的 aliYunAccount.accessKeySecret")
	}
	if strings.TrimSpace(c.SignName) == "" {
		return errors.New("短信配置缺失（SignName 短信签名）：请在管理端「业务配置 → 短信」填写，" +
			"签名需在阿里云短信控制台审核通过，且与 AccessKey 同属一个账号")
	}
	return nil
}

// ValidateTemplate 模板码格式校验（Send 时用；空模板由调用方自己判，这里只判格式）
func ValidateTemplate(templateCode string) error {
	t := strings.TrimSpace(templateCode)
	if t == "" {
		return errors.New("短信模板码为空")
	}
	if !aliyunCodePat.MatchString(t) {
		return errors.Errorf("短信模板码格式异常（应以 SMS_ 开头）：当前值 %s", t)
	}
	return nil
}

// ValidatePhone 手机号格式校验（中国大陆；2026-09-27 加：原来只判空，脏号码会白白发一次请求）
func ValidatePhone(phone string) error {
	p := strings.TrimSpace(phone)
	if p == "" {
		return errors.New("手机号为空")
	}
	if !phonePattern.MatchString(p) {
		return errors.Errorf("手机号格式不正确：%s", MaskPhone(p))
	}
	return nil
}

// clientCache 阿里云客户端缓存（key = 配置指纹；client.go 的 Client 也走这里）
//
// 2026-09-27 加：原实现每次发送都 `dysmsapi.NewClient`（每次新建 HTTP 连接池）。
// 官方 SDK 明确建议复用客户端；以「AK+Secret+Endpoint」为 key，换配置自然换实例，
// 既省连接、又不会"改完配置还在用旧 AK"。
var clientCache sync.Map // string → *dysmsapi.Client

func clientFor(cfg Config) (*dysmsapi.Client, error) {
	endpoint := defaultEndpoint
	key := strings.TrimSpace(cfg.AccessKeyId) + "|" + strings.TrimSpace(cfg.AccessKeySecret) + "|" + endpoint
	if hit, ok := clientCache.Load(key); ok {
		if c, ok := hit.(*dysmsapi.Client); ok && c != nil {
			return c, nil
		}
	}
	c, err := dysmsapi.NewClient(&openapi.Config{
		AccessKeyId:     tea.String(strings.TrimSpace(cfg.AccessKeyId)),
		AccessKeySecret: tea.String(strings.TrimSpace(cfg.AccessKeySecret)),
		Endpoint:        tea.String(endpoint),
		// SDK 层超时（原来只有 ctx 兜底，SDK 内部 HTTP 无超时）。
		// 类型是 *int（毫秒），不是 *int32 —— 传错类型编译不过。
		ConnectTimeout: tea.Int(int(sdkTimeout / time.Millisecond)),
		ReadTimeout:    tea.Int(int(sdkTimeout / time.Millisecond)),
	})
	if err != nil {
		return nil, errors.Wrap(err, "创建短信客户端失败")
	}
	clientCache.Store(key, c)
	return c, nil
}

// SendDetail 结构化发送（2026-09-27 新增，返回阿里云 Code/RequestId/BizId）
//
// Send 是它的薄封装（只保留 error 语义），三个既有调用点不需要改。
func SendDetail(ctx context.Context, cfg Config, phone string, templateCode string, params map[string]string) (SendResult, error) {
	var out SendResult
	if ctx == nil {
		ctx = context.Background()
	}
	// ① 格式/必填校验全在前端（= 本函数内、发出请求之前）——不通过就**不发**，省一次无效调用
	if err := ValidatePhone(phone); err != nil {
		return out, err
	}
	if err := ValidateTemplate(templateCode); err != nil {
		return out, err
	}
	if err := cfg.Validate(); err != nil {
		return out, err
	}
	paramJSON := "{}"
	if len(params) > 0 {
		b, mErr := json.Marshal(params)
		if mErr != nil {
			return out, errors.Wrap(mErr, "sms: 模板参数序列化失败")
		}
		paramJSON = string(b)
	}

	// ② 超时保护：goroutine + select 兜住（与 mailUtils 同口径），与 SDK 层超时双保险
	type result struct {
		code      string
		message   string
		bizId     string
		requestId string
		err       error
	}
	done := make(chan result, 1)
	go func() {
		client, cErr := clientFor(cfg)
		if cErr != nil {
			done <- result{err: cErr}
			return
		}
		resp, sErr := client.SendSms(&dysmsapi.SendSmsRequest{
			PhoneNumbers:  tea.String(strings.TrimSpace(phone)),
			SignName:      tea.String(strings.TrimSpace(cfg.SignName)),
			TemplateCode:  tea.String(strings.TrimSpace(templateCode)),
			TemplateParam: tea.String(paramJSON),
		})
		if sErr != nil {
			// SDK 错误：把 Code/RequestId 尽力取出来（找阿里云工单要 RequestId）
			r := sdkErrInfo(sErr)
			done <- result{code: r.Code, message: r.Message, requestId: r.RequestId, err: sErr}
			return
		}
		if resp == nil || resp.Body == nil {
			done <- result{err: errors.New("短信发送失败：响应为空")}
			return
		}
		done <- result{
			code:      tea.StringValue(resp.Body.Code),
			message:   tea.StringValue(resp.Body.Message),
			bizId:     tea.StringValue(resp.Body.BizId),
			requestId: tea.StringValue(resp.Body.RequestId),
		}
	}()

	select {
	case <-ctx.Done():
		return out, errors.New("sms: 发送超时或被取消")
	case r := <-done:
		out = SendResult{Code: r.code, Message: r.message, BizId: r.bizId, RequestId: r.requestId}
		if r.err != nil {
			// SDK 层错误也带上可读提示（原文仍由调用方进日志）
			if hint := ExplainCode(r.code); hint != "" {
				return out, errors.Errorf("%s（code=%s requestId=%s；原始错误：%v）", hint, orDash(r.code), orDash(r.requestId), r.err)
			}
			return out, errors.Wrap(r.err, "短信 API 调用失败")
		}
		// 业务码非 OK = 发送失败（签名/模板不匹配、手机号黑名单、欠费…）→ 中文可读提示
		if r.code != "OK" {
			hint := ExplainCode(r.code)
			if hint == "" {
				hint = "短信发送失败"
			}
			return out, errors.Errorf("%s（code=%s requestId=%s；阿里云返回：%s）",
				hint, orDash(r.code), orDash(r.requestId), r.message)
		}
		out.Success = true
		return out, nil
	}
}

// Send 按模板发短信（保留原签名与语义；内部走 SendDetail）
//
// params 是模板变量（键名必须与阿里云控制台申请模板时填的一致，否则会被判参数不匹配）。
// ctx 只用于超时控制：阿里云 SDK 走 HTTP，超时即返回错误（调用方按"失败不阻断主流程"处理）。
func Send(ctx context.Context, cfg Config, phone string, templateCode string, params map[string]string) error {
	_, err := SendDetail(ctx, cfg, phone, templateCode, params)
	return err
}

// SendWithTimeout 带默认超时的 Send（懒人版；业务侧一般直接用它）
func SendWithTimeout(cfg Config, phone string, templateCode string, params map[string]string) error {
	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()
	return Send(ctx, cfg, phone, templateCode, params)
}

// SendDetailWithTimeout 带默认超时的 SendDetail（要拿 Code/RequestId 的调用点用这个，
// 例如管理端「测试短信」—— 运营需要 RequestId 找阿里云查）
func SendDetailWithTimeout(cfg Config, phone string, templateCode string, params map[string]string) (SendResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()
	return SendDetail(ctx, cfg, phone, templateCode, params)
}

// MaskPhone 手机号打码（日志用，避免把完整号码写进日志）
func MaskPhone(phone string) string {
	if len(phone) <= 4 {
		return "****"
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}

// MaskSecret 凭据打码（日志/错误里用；**只露后 4 位**，不输出完整 AK）
func MaskSecret(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "(空)"
	}
	if len(s) <= 4 {
		return "****"
	}
	return "****" + s[len(s)-4:]
}

// orDash 空串转 "-"，避免日志里出现 `code= requestId=`
func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

// FormatTemplateParam 辅助：把 map 拼成阿里云要求的 JSON 字符串（调试/日志用）
func FormatTemplateParam(params map[string]string) string {
	if len(params) == 0 {
		return "{}"
	}
	b, err := json.Marshal(params)
	if err != nil {
		return "{}"
	}
	return string(b)
}
