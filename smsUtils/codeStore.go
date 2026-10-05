// Package smsUtils 的验证码存储部分：CodeStore 接口 + 两种实现（Redis / 本地内存）。
//
// 为什么把"验证码存储"做成接口而不是直接用 redis.Client（2026-09-26）：
//   - 单机/测试/离线环境没有 Redis，但验证码功能仍要能跑通（不能"没 Redis 就发不出码"）；
//   - 多副本部署**必须**用 Redis（local 不跨实例：发码在 A 实例、校验在 B 实例必然失败），
//     所以两种模式都要有，且**默认必须是 redis**；
//   - 业务侧（验证码服务）只面向接口编程，换存储不需要改业务代码。
//
// 接口语义（抽自 xianyu-open 的 sms.RedisCodeStore，两处按本项目需要调整并在注释里写明）：
//
//	① 参数叫 key 而不是 phone：验证码的"目标"早已泛化（手机号 / 邮箱 / 账号 id 等），
//	   业务侧拼好完整 key（如 "email:foo@bar.com"）传进来，存储只负责加前缀与读写；
//	② 新增 Clear：发送通道失败时业务侧要回滚"已占的频控 + 已落的码"，让用户能立即重试。
//	   （参考项目是"发成功才落码"，但那样在并发下会出现"用户收到码却校验失败"。）
package smsUtils

import (
	"context"
	crand "crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
)

// 存储模式取值（StoreConfig.Mode）
const (
	// StoreModeRedis 分布式存储（**默认**，多副本部署必须用它）
	StoreModeRedis = "redis"
	// StoreModeLocal 进程内内存（单实例 / 测试 / 无 Redis 的离线环境）
	StoreModeLocal = "local"
)

// StoreConfig 验证码存储配置（由调用方从自己的配置体系装配，本包不读任何配置文件）
type StoreConfig struct {
	// Mode 存储模式：StoreModeRedis（默认）/ StoreModeLocal
	Mode string `json:"mode"`
	// Redis Mode=redis 时必填（由调用方注入，公共库不自己造连接）
	Redis *redis.Client
	// KeyPrefix 验证码本体 key 前缀（如 "captcha:" → 最终 key = "captcha:" + 业务 key）
	KeyPrefix string `json:"keyPrefix"`
	// LimitKeyPrefix 发送频控 key 前缀（如 "captcha:limit:"）
	LimitKeyPrefix string `json:"limitKeyPrefix"`
	// CodeTTL 验证码有效期
	CodeTTL time.Duration `json:"codeTtl"`
	// SendInterval 同一目标的最���发送间隔（频控）
	SendInterval time.Duration `json:"sendInterval"`
	// MaxAttempts 同一验证码最多允许比对失败几次；**0 = 不限制**（不启用计数 key）
	MaxAttempts int `json:"maxAttempts"`
}

// 缺省值（调用方没配时的兜底，避免"忘了配 TTL 导致验证码永不过期"这类事故）
const (
	defaultCodeTTL      = 10 * time.Minute
	defaultSendInterval = 60 * time.Second
	defaultKeyPrefix    = "sms:code:"
	defaultLimitPrefix  = "sms:send_limit:"
)

// withDefaults 补齐缺省值（不修改入参，返回副本）
func (c StoreConfig) withDefaults() StoreConfig {
	if c.KeyPrefix == "" {
		c.KeyPrefix = defaultKeyPrefix
	}
	if c.LimitKeyPrefix == "" {
		c.LimitKeyPrefix = defaultLimitPrefix
	}
	if c.CodeTTL <= 0 {
		c.CodeTTL = defaultCodeTTL
	}
	if c.SendInterval <= 0 {
		c.SendInterval = defaultSendInterval
	}
	return c
}

// CodeStore 验证码存储接口
//
// 四个方法对应验证码的完整生命周期：频控检查 → 记录发送 → 落码 → 校验。
// 通用性说明：**短信/邮箱/其它渠道**都只是"key 的构造方式"不同，存储本身不关心。
type CodeStore interface {
	// CanSend 检查当前是否允许发送（频控）；(false, msg) 时 msg 形如"请 42 秒后再试"，
	// 业务侧可直接把它透给前端做倒计时（前端约定从文案里正则取秒数）。
	CanSend(ctx context.Context, key string) (bool, string)
	// RecordSend 记录一次发送（写频控占位）
	RecordSend(ctx context.Context, key string) error
	// Store 落码（覆盖旧码）
	Store(ctx context.Context, key, code string) error
	// Verify 校验；**通过即删除**（一次性，防重放）。(false, msg) 为校验失败原因。
	Verify(ctx context.Context, key, code string) (bool, string)
	// Clear 清掉该 key 的验证码与频控占位（发送通道失败时回滚，让用户能立即重试）
	Clear(ctx context.Context, key string) error
}

// NewCodeStore 按配置创建存储实现（照本库 lockUtils.NewLocker 的范式：
// 依赖由调用方注入、工厂只做选择；配置非法直接 panic——那是启动期编程错误，早炸好过运行期诡异行为）
func NewCodeStore(cfg StoreConfig) CodeStore {
	cfg = cfg.withDefaults()
	switch cfg.Mode {
	case StoreModeLocal, "":
		return NewLocalCodeStore(cfg)
	case StoreModeRedis:
		if cfg.Redis == nil {
			panic("smsUtils.NewCodeStore: Mode=redis 但未注入 Redis 客户端")
		}
		return NewRedisCodeStore(cfg)
	default:
		panic(fmt.Sprintf("smsUtils.NewCodeStore: 未知的存储模式 %q（支持 %s / %s）",
			cfg.Mode, StoreModeRedis, StoreModeLocal))
	}
}

// GenerateCode 生成 6 位数字验证码（crypto/rand，前导零保留）
//
// crypto/rand 失败是系统级异常（熵源不可用），此时返回错误让业务侧记日志，
// **不**像某些实现那样 fallback 成 "000000"——那会让所有人拿到同一个码。
func GenerateCode() (string, error) {
	n, err := crand.Int(crand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", errors.Wrap(err, "生成验证码失败")
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
