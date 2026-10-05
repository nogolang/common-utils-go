// Package mailUtils 通用发信（SMTP）
//
// 2026-09-26 从 shop-common/commonUtil/mailUtils 迁入公共库。
//
// 配置口径：**只认传进来的结构体，不读任何配置文件**（与 gormUtils / redisUtils 同款）。
// 主机/端口/账号/授权码的来源由调用方决定——本项目（3-my-shop-single）从业务配置
// （sys_business_config）装配。迁入前本包的 LoadConfig 走 viper 直读，已删除。
//
// ⚠️ 三条纪律：
//  1. **必须带超时**：老实现直接 `DialAndSend` 无超时，SMTP 服务器卡住会把调用方 goroutine 挂死
//     （在 HTTP 请求路径上就是接口悬挂）。这里统一 10s（可传 ctx 提前取消）。
//  2. **调用方必须容错**：发信失败绝不能让业务主流程失败——通知类邮件失败只记日志
//     （见 shop-server-admin noticeSendHandler 的注释：返回 error 会触发 MQ 重试 → 重复发信）。
//  3. **AuthCode 是授权码不是登录密码**：QQ 邮箱等要求 SMTP 授权码。
package mailUtils

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/mail.v2"
)

// dialTimeout 单次投递超时（含 TLS 握手 + SMTP 会话）；SMTP 是外部依赖，不能无限等
const dialTimeout = 10 * time.Second

// Config 发信配置，**全部由调用方注入**
type Config struct {
	// Host SMTP 服务器（如 smtp.qq.com）
	Host string
	// Port 端口（隐式 SSL 一般 465）
	Port int
	// User 发件邮箱（同时是 From 地址）
	User string
	// AuthCode SMTP 授权码（**授权码**，不是登录密码）
	AuthCode string
}

// Validate 必填校验；缺失返回可读错误（调用方按"没配好就不发"处理）。
func (c Config) Validate() error {
	if strings.TrimSpace(c.Host) == "" || strings.TrimSpace(c.User) == "" || strings.TrimSpace(c.AuthCode) == "" {
		return errors.New("smtp 配置缺失（Host / User / AuthCode）")
	}
	return nil
}

// emailRe 常规邮箱格式
var emailRe = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// ValidEmail 邮箱格式校验（写邮箱前的统一口径）
//
// 放这里而不是各服务各写一份：用户端验证码链路与商家端「改通知邮箱」都要用，
// 两处口径不一致会出现"验证码能发、通知邮箱存不进去"这种怪现象。
func ValidEmail(email string) bool {
	return emailRe.MatchString(strings.TrimSpace(email))
}

// Send 发一封 HTML 邮件
//
// senderName 是发件人显示名（如站点名），会由 mail.v2 做 RFC2047 编码，可放心用中文。
// 465 端口走隐式 SSL（先 TLS 握手再 SMTP）；若换成 587/25 明文端口，请把 SSL 关掉并开 STARTTLS。
func Send(ctx context.Context, cfg Config, to, subject, htmlBody, senderName string) error {
	to = strings.TrimSpace(to)
	if to == "" {
		return errors.New("收件人为空")
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	m := mail.NewMessage()
	m.SetAddressHeader("From", cfg.User, senderName)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", htmlBody)

	d := mail.NewDialer(cfg.Host, cfg.Port, cfg.User, cfg.AuthCode)
	d.SSL = true
	d.Timeout = dialTimeout
	// ctx 取消 → 尽快放弃（DialAndSend 不支持 ctx，用一个带缓冲的 chan 在返回时释放 goroutine）
	done := make(chan error, 1)
	go func() { done <- d.DialAndSend(m) }()
	select {
	case err := <-done:
		if err != nil {
			return errors.Wrap(err, "SMTP 投递失败")
		}
		return nil
	case <-ctx.Done():
		return errors.Wrap(ctx.Err(), "SMTP 投递被取消")
	case <-time.After(dialTimeout + 2*time.Second):
		// 双保险：dialer 自身的 Timeout 之外再加一层，防止实现细节变化导致悬挂
		return fmt.Errorf("SMTP 投递超时（%s）", dialTimeout)
	}
}
