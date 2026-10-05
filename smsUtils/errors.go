package smsUtils

import (
	"regexp"
	"strings"

	"github.com/alibabacloud-go/tea/tea"
)

// ============================================================================
// 阿里云短信错误码 → 可读中文（2026-09-27）
// ----------------------------------------------------------------------------
// 为什么要这张表：阿里云返回的是 `InvalidAccessKeyId.NotFound` / `isv.SMS_SIGNATURE_ILLEGAL`
// 这类**运维看不懂、也没法自己解决**的码。2026-09-26 实测的现场：配置检查明明打印
// 「短信AK就绪: true」，点测试短信却只看到 `Specified access key is not found` ——
// 既不知道是哪个 AK、也不知道该改哪一行配置。
//
// 纪律：**阿里云原始 message 只进日志**，前端/弹窗只拿这里的可读提示
//（原始 message 里含 requestId 等内部信息，前端不需要也不该展示）。
// 未收录的码走 ExplainCode 的兜底分支，保证"永远给得出一句人话"。
// ============================================================================

// codeHints 阿里云短信错误码 → 可读提示
//
// 覆盖来源：阿里云 dysmsapi 官方错误码表里与本项目部署形态相关的部分
// （短信业务错误以 isv. 前缀，账号/网络类以 InvalidAccessKeyId. 等前缀）。
var codeHints = map[string]string{
	// ---------- 账号 / 凭据（本项目最常踩的一类）----------
	"InvalidAccessKeyId.NotFound":          "阿里云 AccessKeyId 无效或已停用：请检查 common.*.yaml 的 aliYunAccount.accessKeyId（与 OSS 共用同一账号），改后需重启服务",
	"InvalidAccessKeyId.Secret":            "阿里云 AccessKeySecret 与 AccessKeyId 不匹配：请检查 common.*.yaml 的 aliYunAccount.accessKeySecret（注意别把 Id 与 Secret 填反）",
	"InvalidParameter.AccessKeyIdNotFound": "阿里云 AccessKeyId 无效或已停用：请检查 common.*.yaml 的 aliYunAccount.accessKeyId",

	// ---------- 签名 / 模板（本项目次常踩）----------
	"isv.SMS_SIGNATURE_ILLEGAL":             "短信签名不存在或未通过审核：请在阿里云短信控制台确认该签名的**审核状态为已通过**，且与 AccessKey 属同一账号",
	"isv.SMS_SIGNATURE_NOT_ET_NEED_APPROVE": "短信签名未通过审核：请在阿里云短信控制台签名列表里确认审核状态",
	"isv.SMS_TEMPLATE_ILLEGAL":              "短信模板码不存在或未通过审核：请在管理端「业务配置 → 短信」核对模板码（形如 SMS_xxxx），并确认它与 AccessKey 属同一账号",
	"isv.SMS_TEMPLATE_NOT_ET_NEED_APPROVE":  "短信模板未通过审核：请在阿里云短信控制台模板列表里确认审核状态",
	"isv.TEMPLATE_NOT_EXIST":                "短信模板不存在：模板码可能被删除或属于另一个阿里云账号",

	// ---------- 账号状态 / 余额 ----------
	"isv.OUT_OF_SERVICE":          "阿里云短信服务已停用（通常是**账号欠费**）：请在阿里云控制台充值或开通短信服务",
	"isv.BDAY_BALANCE_NOT_ENOUGH": "阿里云短信余额不足：请在阿里云控制台充值",
	"isv.SMS_ACCOUNT_ABNORMAL":    "阿里云短信账号状态异常：请在阿里云控制台确认账号是否正常（欠费/违规停服）",

	// ---------- 限流 / 频控 ----------
	"isv.BUSINESS_LIMIT_CONTROL": "触发阿里云短信限流（流控：同一手机号/内容发送过快或单日量超限）：请稍后重试，或在阿里云控制台调整频控",
	"Throttling.System":          "阿里云接口被限流（请求过快）：请稍后重试",
	"Throttling.User":            "阿里云账号被限流（请求过快）：请稍后重试",

	// ---------- 手机号 ----------
	"isv.MOBILE_NUMBER_ILLEGAL":             "手机号格式不正确（需 11 位大陆号码）",
	"isv.MOBILE_COUNT_OVER_LIMIT":           "该手机号当日发送量已达上限：被阿里云频控拦下，请稍后再试",
	"isv.MOBILE_NUMBER_ILLEGAL_OR_USER_UNL": "手机号格式不正确，或该号码在阿里云黑名单中",

	// ---------- 内容 / 其它 ----------
	"isv.CONTENT_NOT_ET_NEED_APPROVE": "短信内容未通过审核（含敏感词）：请调整短信内容后重试",
	"isv.INTERFACE_LIMIT":             "阿里云接口调用超限（QPS/日调用量）：请稍后重试",
}

// ExplainCode 把阿里云错误码翻成可读中文（未收录时按前缀兜底）
//
// 兜底分三种：
//   - 含 Signature/Template → 引导去核对签名/模板（这类码最常见，值得模糊匹配）
//   - 含 Limit/Throttling  → 提示限流
//   - 其它                 → 如实说"阿里云拒绝"，并把 code 带上让人去查官方文档
func ExplainCode(code string) string {
	c := strings.TrimSpace(code)
	if c == "" || c == "OK" {
		return ""
	}
	if hint, ok := codeHints[c]; ok {
		return hint
	}
	lower := strings.ToLower(c)
	switch {
	case strings.Contains(lower, "signature"):
		return "短信签名相关问题（不存在/未审核/与账号不匹配）：请在阿里云短信控制台核对该签名的审核状态与所属账号"
	case strings.Contains(lower, "template"):
		return "短信模板相关问题（不存在/未审核/与账号不匹配）：请核对管理端「业务配置 → 短信」里的模板码，并确认它与 AccessKey 属同一阿里云账号"
	case strings.Contains(lower, "limit"), strings.Contains(lower, "throttl"):
		return "触发阿里云短信限流（发送过快或单日量超限）：请稍后重试"
	case strings.Contains(lower, "accesskey"):
		return "阿里云 AccessKey 相关问题（无效/不匹配/已停用）：请检查 common.*.yaml 的 aliYunAccount 段（与 OSS 共用同一账号），改后需重启服务"
	default:
		return "阿里云拒绝了本次短信请求（详见 code 与服务端日志里的原始返回）"
	}
}

// sdkErrInfo 从 SDK 错误里尽力取出 Code / Message / RequestId
//
// 为什么需要：SDK 返回的是 `*tea.SDKError`，Code/Message 都是 *string；
// **它没有 RequestId 字段** —— 而 RequestId 恰恰是找阿里云工单排查的唯一凭据。
// 传输层错误的 request id 藏在 message 文本里（形如
// `code: 404, Specified access key is not found. request id: 01A0DE69-...`），
// 这里用正则把它抠出来，业务失败（resp 非 nil）时由调用方从 resp.Body.RequestId 直接取。
func sdkErrInfo(err error) (info struct{ Code, Message, RequestId string }) {
	se, ok := err.(*tea.SDKError)
	if !ok {
		return info
	}
	info.Code = tea.StringValue(se.Code)
	info.Message = tea.StringValue(se.Message)
	info.RequestId = requestIdFromMessage(info.Message)
	return info
}

// requestIdPat 阿里云错误 message 里的 request id（形如 `request id: 01A0DE69-...`）
var requestIdPat = regexp.MustCompile(`(?i)request\s*id\s*[:=]\s*([0-9A-Za-z-]{8,})`)

// requestIdFromMessage 从错误文本里抠出阿里云 request id（抠不到返回空串）
func requestIdFromMessage(msg string) string {
	if m := requestIdPat.FindStringSubmatch(msg); len(m) == 2 {
		return m[1]
	}
	return ""
}
