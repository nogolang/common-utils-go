package smsUtils

// Redis 版验证码存储（多实例共享，集群安全）
//
// key 口径全部由 StoreConfig 决定，**实现里不写死任何业务前缀**：
// 本项目传 KeyPrefix="captcha:"、业务 key 传 "email:{邮箱}"，最终就是
// `captcha:email:{邮箱}`；参考项目传 "sms:code:"、key 传手机号，就是 `sms:code:{手机号}`。
// 换前缀不用改实现，改存量口径会作废线上已有验证码与所有直写 key 的冒烟脚本——慎改。
import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// RedisCodeStore 基于 Redis 的验证码存储
type RedisCodeStore struct {
	rdb *redis.Client
	cfg StoreConfig
}

// NewRedisCodeStore 创建 Redis 验证码存储（cfg 会先补齐缺省值）
func NewRedisCodeStore(cfg StoreConfig) *RedisCodeStore {
	return &RedisCodeStore{rdb: cfg.Redis, cfg: cfg.withDefaults()}
}

// 内部 key 构造（业务 key 前缀化；频控 key 另用 LimitKeyPrefix）
func (s *RedisCodeStore) codeKey(key string) string { return s.cfg.KeyPrefix + key }
func (s *RedisCodeStore) limitKey(key string) string {
	return s.cfg.LimitKeyPrefix + key
}
func (s *RedisCodeStore) attemptsKey(key string) string { return s.codeKey(key) + ":attempts" }

// CanSend 频控检查：占位存在即不允许，并把**剩余秒数**写进提示（前端据此显示倒计时）
func (s *RedisCodeStore) CanSend(ctx context.Context, key string) (bool, string) {
	exists, err := s.rdb.Exists(ctx, s.limitKey(key)).Result()
	if err != nil {
		// Redis 故障：fail-open 放行（与本项目原实现同款取舍——可用性优先，
		// 真正落码那一步还会再暴露故障）
		return true, ""
	}
	if exists > 0 {
		ttl, _ := s.rdb.TTL(ctx, s.limitKey(key)).Result()
		remaining := int(ttl.Seconds())
		if remaining <= 0 {
			remaining = int(s.cfg.SendInterval.Seconds())
		}
		return false, fmt.Sprintf("发送过于频繁，请 %d 秒后再试", remaining)
	}
	return true, ""
}

// RecordSend 写频控占位（SETNX + TTL；抢不到说明并发下别人刚发过，业务侧自行决定是否忽略）
func (s *RedisCodeStore) RecordSend(ctx context.Context, key string) error {
	return s.rdb.SetNX(ctx, s.limitKey(key), "1", s.cfg.SendInterval).Err()
}

// Store 落码（覆盖旧码；启用次数上限时一并把计数清零）
func (s *RedisCodeStore) Store(ctx context.Context, key, code string) error {
	pipe := s.rdb.Pipeline()
	pipe.Set(ctx, s.codeKey(key), code, s.cfg.CodeTTL)
	if s.cfg.MaxAttempts > 0 {
		pipe.Set(ctx, s.attemptsKey(key), 0, s.cfg.CodeTTL)
	}
	_, err := pipe.Exec(ctx)
	return err
}

// Verify 校验；通过即删（一次性）
func (s *RedisCodeStore) Verify(ctx context.Context, key, code string) (bool, string) {
	if code == "" {
		return false, "验证码不能为空"
	}
	stored, err := s.rdb.Get(ctx, s.codeKey(key)).Result()
	if err != nil {
		// key 不存在（未发送 / 已过期 / 已用过）与"码不对"**同口径**，不区分提示防探测
		if err == redis.Nil {
			return false, "验证码错误或已过期"
		}
		return false, "系统错误，请稍后再试"
	}
	if s.cfg.MaxAttempts > 0 {
		attempts, _ := s.rdb.Get(ctx, s.attemptsKey(key)).Int()
		if attempts >= s.cfg.MaxAttempts {
			// 错太多次：连码一起清掉，强制重新获取
			s.rdb.Del(ctx, s.codeKey(key), s.attemptsKey(key))
			return false, "验证码错误次数过多，请重新获取"
		}
		s.rdb.Incr(ctx, s.attemptsKey(key))
	}
	if stored != code {
		return false, "验证码错误"
	}
	if err := s.rdb.Del(ctx, s.codeKey(key)).Err(); err != nil {
		// 删失败不影响本次校验通过（下次校验会因为码还在而失败一次，代价可接受）
		return true, ""
	}
	return true, ""
}

// Clear 回滚：清验证码 + 频控占位（发送通道失败时调用，让用户能立即重试）
func (s *RedisCodeStore) Clear(ctx context.Context, key string) error {
	keys := []string{s.codeKey(key), s.limitKey(key)}
	if s.cfg.MaxAttempts > 0 {
		keys = append(keys, s.attemptsKey(key))
	}
	return s.rdb.Del(ctx, keys...).Err()
}

// compile-time 断言：实现必须满足接口（少一个方法就是运行期才发现）
var _ CodeStore = (*RedisCodeStore)(nil)
