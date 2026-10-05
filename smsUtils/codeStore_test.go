package smsUtils

// 验证码存储（CodeStore）的单测 —— 2026-09-26
//
// 覆盖三件最容易出错的事：
//  ① **key 口径**（Redis 版）：本项目线上是 `captcha:email:{邮箱}` / `captcha:limit:email:{邮箱}`，
//     线上存量验证码与 4 个冒烟脚本都直写这个 key —— 口径一变就全线失效，必须钉死；
//  ② **一次性**语义：校验通过即删（防重放），这是验证码的安全底线；
//  ③ **local 实现**：过期、一次性、频控剩余秒数文案（前端倒计时靠它）。
//
// Redis 版用 miniredis 起内存 Redis（不依赖本机 dev Redis，CI 也能跑）。

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// ==================== key 口径（最容易回归的一条） ====================

func TestRedisCodeStore_KeyMatchesProjectContract(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	// 本项目的装配口径（见 CaptchaSvc 的 Store 注入）
	store := NewRedisCodeStore(StoreConfig{
		Mode:           StoreModeRedis,
		Redis:          rdb,
		KeyPrefix:      "captcha:",
		LimitKeyPrefix: "captcha:limit:",
		CodeTTL:        10 * time.Minute,
		SendInterval:   60 * time.Second,
	})
	ctx := context.Background()
	key := "email:foo@bar.com"

	if err := store.RecordSend(ctx, key); err != nil {
		t.Fatalf("RecordSend 失败: %v", err)
	}
	if err := store.Store(ctx, key, "123456"); err != nil {
		t.Fatalf("Store 失败: %v", err)
	}

	keys := mr.Keys()
	hasCode, hasLimit := false, false
	for _, k := range keys {
		if k == "captcha:email:foo@bar.com" {
			hasCode = true
		}
		if k == "captcha:limit:email:foo@bar.com" {
			hasLimit = true
		}
	}
	if !hasCode {
		t.Fatalf("验证码 key 口径变了：期望 captcha:email:foo@bar.com，实际 %v", keys)
	}
	if !hasLimit {
		t.Fatalf("频控 key 口径变了：期望 captcha:limit:email:foo@bar.com，实际 %v", keys)
	}
}

// 换前缀不应该残留默认前缀（防止调用方漏传前缀时悄悄写到 sms:code: 去）
func TestRedisCodeStore_PrefixIsFullyConfigurable(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store := NewRedisCodeStore(StoreConfig{Redis: rdb, KeyPrefix: "myapp:otp:", LimitKeyPrefix: "myapp:otp:gap:"})
	ctx := context.Background()
	_ = store.RecordSend(ctx, "phone:13800000000")
	_ = store.Store(ctx, "phone:13800000000", "654321")
	for _, k := range mr.Keys() {
		if !strings.HasPrefix(k, "myapp:otp:") {
			t.Fatalf("出现了非配置前缀的 key: %s", k)
		}
	}
}

// ==================== 通用行为（两种实现各跑一遍） ====================

// storeFixture 造一个"跑完即清理"的存储；redis 模式用内存 Redis
func storeFixture(t *testing.T, mode string) (CodeStore, *miniredis.Miniredis) {
	t.Helper()
	cfg := StoreConfig{
		Mode:           mode,
		KeyPrefix:      "captcha:",
		LimitKeyPrefix: "captcha:limit:",
		CodeTTL:        10 * time.Minute,
		SendInterval:   60 * time.Second,
	}
	var mr *miniredis.Miniredis
	if mode == StoreModeRedis {
		mr = miniredis.RunT(t)
		cfg.Redis = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	}
	return NewCodeStore(cfg), mr
}

func TestCodeStore_Lifecycle(t *testing.T) {
	for _, mode := range []string{StoreModeLocal, StoreModeRedis} {
		t.Run(mode, func(t *testing.T) {
			store, _ := storeFixture(t, mode)
			ctx := context.Background()
			key := "phone:13800000000"

			// ① 没发过码就能发
			if ok, msg := store.CanSend(ctx, key); !ok {
				t.Fatalf("首次应可发送，实际被拒: %s", msg)
			}
			// ② 记录发送后进入频控，且文案带剩余秒数（前端倒计时的数据源）
			if err := store.RecordSend(ctx, key); err != nil {
				t.Fatalf("RecordSend: %v", err)
			}
			ok, msg := store.CanSend(ctx, key)
			if ok {
				t.Fatal("发过一次后应被频控拦住")
			}
			if !strings.Contains(msg, "秒后再试") {
				t.Fatalf("频控文案应带剩余秒数（前端靠它倒计时），实际: %q", msg)
			}
			// ③ 落码 → 错码不过
			if err := store.Store(ctx, key, "123456"); err != nil {
				t.Fatalf("Store: %v", err)
			}
			if ok, _ := store.Verify(ctx, key, "000000"); ok {
				t.Fatal("错误验证码不应通过")
			}
			// ④ 对码通过
			if ok, msg := store.Verify(ctx, key, "123456"); !ok {
				t.Fatalf("正确验证码应通过，实际: %s", msg)
			}
			// ⑤ **一次性**：通过后立刻失效（防重放）
			if ok, _ := store.Verify(ctx, key, "123456"); ok {
				t.Fatal("验证码必须一次性：通过后再次校验应失败")
			}
		})
	}
}

func TestCodeStore_ClearRollsBack(t *testing.T) {
	for _, mode := range []string{StoreModeLocal, StoreModeRedis} {
		t.Run(mode, func(t *testing.T) {
			store, _ := storeFixture(t, mode)
			ctx := context.Background()
			key := "email:foo@bar.com"

			_ = store.RecordSend(ctx, key)
			_ = store.Store(ctx, key, "123456")
			// 发送通道失败 → 回滚：码与频控都要清掉，用户能立即重试
			if err := store.Clear(ctx, key); err != nil {
				t.Fatalf("Clear: %v", err)
			}
			if ok, _ := store.CanSend(ctx, key); !ok {
				t.Fatal("回滚后不应再被频控拦住")
			}
			if ok, _ := store.Verify(ctx, key, "123456"); ok {
				t.Fatal("回滚后验证码应失效")
			}
		})
	}
}

func TestCodeStore_EmptyCodeRejected(t *testing.T) {
	store, _ := storeFixture(t, StoreModeLocal)
	if ok, _ := store.Verify(context.Background(), "phone:138", ""); ok {
		t.Fatal("空验证码不应通过")
	}
}

// ==================== local 特有行为 ====================

func TestLocalCodeStore_Expires(t *testing.T) {
	store := NewLocalCodeStore(StoreConfig{CodeTTL: 50 * time.Millisecond, SendInterval: time.Minute})
	ctx := context.Background()
	key := "phone:13800000000"
	_ = store.Store(ctx, key, "123456")
	if ok, _ := store.Verify(ctx, key, "123456"); !ok {
		t.Fatal("未过期时校验应通过")
	}

	_ = store.Store(ctx, key, "123456")
	time.Sleep(80 * time.Millisecond) // 超过 TTL
	if ok, msg := store.Verify(ctx, key, "123456"); ok {
		t.Fatal("过期验证码不应通过")
	} else if !strings.Contains(msg, "过期") {
		t.Fatalf("过期提示文案不对: %q", msg)
	}
	// 过期条目应被惰性清掉，不留残渣
	if codes, _ := store.Len(); codes != 0 {
		t.Fatalf("过期条目应被惰性清理，剩余 %d 条", codes)
	}
}

func TestLocalCodeStore_SendGapExpires(t *testing.T) {
	store := NewLocalCodeStore(StoreConfig{CodeTTL: time.Minute, SendInterval: 50 * time.Millisecond})
	ctx := context.Background()
	_ = store.RecordSend(ctx, "phone:13800000000")
	if ok, _ := store.CanSend(ctx, "phone:13800000000"); ok {
		t.Fatal("频控期内应被拦")
	}
	time.Sleep(80 * time.Millisecond)
	if ok, _ := store.CanSend(ctx, "phone:13800000000"); !ok {
		t.Fatal("频控过期后应放行")
	}
}

func TestLocalCodeStore_Concurrent(t *testing.T) {
	store := NewLocalCodeStore(StoreConfig{CodeTTL: time.Minute, SendInterval: 50 * time.Millisecond})
	ctx := context.Background()
	key := "phone:13800000000"
	_ = store.Store(ctx, key, "123456")

	// 并发校验：只有一个 goroutine 能拿到"通过"
	passCh := make(chan bool, 32)
	for i := 0; i < 32; i++ {
		go func() {
			ok, _ := store.Verify(ctx, key, "123456")
			passCh <- ok
		}()
	}
	pass := 0
	for i := 0; i < 32; i++ {
		if <-passCh {
			pass++
		}
	}
	if pass != 1 {
		t.Fatalf("并发校验下只有 1 个应通过，实际 %d 个通过（一次性语义被击穿）", pass)
	}
}

// ==================== 工厂 ====================

func TestNewCodeStore_Modes(t *testing.T) {
	if _, ok := NewCodeStore(StoreConfig{Mode: StoreModeLocal}).(*LocalCodeStore); !ok {
		t.Fatal("Mode=local 应返回 LocalCodeStore")
	}
	// Mode 留空 = local（安全默认：没显式说用 Redis 时不该假设有 Redis）
	if _, ok := NewCodeStore(StoreConfig{}).(*LocalCodeStore); !ok {
		t.Fatal("Mode 留空应回落到 local（安全默认）")
	}
	mr := miniredis.RunT(t)
	if _, ok := NewCodeStore(StoreConfig{Mode: StoreModeRedis,
		Redis: redis.NewClient(&redis.Options{Addr: mr.Addr()})}).(*RedisCodeStore); !ok {
		t.Fatal("Mode=redis 应返回 RedisCodeStore")
	}
}

// 配置错误必须在**启动期**暴露（panic），而不是运行期第一次发码才发现
func TestNewCodeStore_PanicsOnBadConfig(t *testing.T) {
	assertPanic(t, "Mode=redis 但没注入 Redis 客户端", func() {
		NewCodeStore(StoreConfig{Mode: StoreModeRedis})
	})
	assertPanic(t, "未知 Mode", func() {
		NewCodeStore(StoreConfig{Mode: "mysql"})
	})
}

func assertPanic(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("%s：期望 panic（配置错误应在启动期暴露）", name)
		}
	}()
	fn()
}

func TestGenerateCode(t *testing.T) {
	for i := 0; i < 50; i++ {
		code, err := GenerateCode()
		if err != nil {
			t.Fatalf("GenerateCode: %v", err)
		}
		if len(code) != 6 {
			t.Fatalf("验证码应为 6 位，实际 %q", code)
		}
		for _, r := range code {
			if r < '0' || r > '9' {
				t.Fatalf("验证码应全为数字，实际 %q", code)
			}
		}
	}
}

func TestStoreConfig_Defaults(t *testing.T) {
	got := StoreConfig{}.withDefaults()
	if got.CodeTTL != defaultCodeTTL || got.SendInterval != defaultSendInterval {
		t.Fatalf("缺省 TTL/频控未补齐: %+v", got)
	}
	if got.KeyPrefix != defaultKeyPrefix || got.LimitKeyPrefix != defaultLimitPrefix {
		t.Fatalf("缺省前缀未补齐: %+v", got)
	}
	// 显式配置不能被缺省值覆盖
	custom := StoreConfig{CodeTTL: time.Minute, KeyPrefix: "captcha:"}.withDefaults()
	if custom.CodeTTL != time.Minute || custom.KeyPrefix != "captcha:" {
		t.Fatalf("显式配置被缺省值覆盖: %+v", custom)
	}
}
