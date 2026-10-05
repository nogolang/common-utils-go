package smsUtils

// 本地内存版验证码存储（单实例 / 测试 / 无 Redis 的离线环境）
//
// ⚠️ **它不跨进程、也不跨副本**——多副本部署下"发码在 A 实例、用户在 B 实例校验"必然失败。
//   所以默认模式是 redis（见 NewCodeStore），local 只在明确单实例时选用。
//   这一点在业务侧配置注释里也写死了，别把默认改成 local。
//
// 过期策略：**惰性过期**（读写时判断 TTL）+ 可选后台清理。
//   为什么不用"起 goroutine 定期扫"：每个实例一个后台协程，长跑进程里要额外管它的生命周期；
//   验证码这种量级（每 key 一个 entry，TTL 10 分钟）惰性清理完全够——过期条目在下次访问时被顺手删掉。
//   真要主动清理，用 SetCleanupInterval（会起一个 goroutine，调用方负责在退出时 Stop）。
import (
	"context"
	"fmt"
	"sync"
	"time"
)

// LocalCodeStore 内存实现（线程安全）
type LocalCodeStore struct {
	mu    sync.Mutex
	codes map[string]localEntry // 验证码本体
	limit map[string]time.Time  // 频控占位的到期时间
	cfg   StoreConfig

	stopClean chan struct{} // 非 nil 表示起了后台清理，由 Stop 负责关
	stopOnce  sync.Once
}

// localEntry 一条验证码（带过期时间，便于惰性判断）
type localEntry struct {
	code    string
	expireA time.Time
}

// NewLocalCodeStore 创建内存验证码存储
func NewLocalCodeStore(cfg StoreConfig) *LocalCodeStore {
	return &LocalCodeStore{
		codes: make(map[string]localEntry),
		limit: make(map[string]time.Time),
		cfg:   cfg.withDefaults(),
	}
}

// CanSend 频控检查（剩余秒数算法与 Redis 版一致，前端倒计时不用区分后端）
func (s *LocalCodeStore) CanSend(ctx context.Context, key string) (bool, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	expireA, ok := s.limit[key]
	if !ok {
		return true, ""
	}
	if remaining := time.Until(expireA); remaining > 0 {
		return false, fmt.Sprintf("发送过于频繁，请 %d 秒后再试", int(remaining.Seconds()))
	}
	// 已过期：顺手清掉
	delete(s.limit, key)
	return true, ""
}

// RecordSend 写频控占位
func (s *LocalCodeStore) RecordSend(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.limit[key] = time.Now().Add(s.cfg.SendInterval)
	return nil
}

// Store 落码（覆盖旧码）
func (s *LocalCodeStore) Store(ctx context.Context, key, code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.codes[key] = localEntry{code: code, expireA: time.Now().Add(s.cfg.CodeTTL)}
	return nil
}

// Verify 校验；通过即删（一次性，与 Redis 版语义一致）
func (s *LocalCodeStore) Verify(ctx context.Context, key, code string) (bool, string) {
	if code == "" {
		return false, "验证码不能为空"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.codes[key]
	if !ok {
		// 未发送 / 已用过 / 已过期同口径，不区分提示防探测
		return false, "验证码错误或已过期"
	}
	if time.Now().After(entry.expireA) {
		delete(s.codes, key)
		return false, "验证码错误或已过期"
	}
	if entry.code != code {
		return false, "验证码错误"
	}
	delete(s.codes, key)
	return true, ""
}

// Clear 回滚：清验证码 + 频控占位
func (s *LocalCodeStore) Clear(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.codes, key)
	delete(s.limit, key)
	return nil
}

// SetCleanupInterval 起一个后台协程定期清理过期条目（默认**不启用**，惰性清理已够）
//
// 什么时候才需要它：单实例内存里验证码流量极大（每秒数千次发码）且长期运行不重启时，
// 过期条目堆积占内存。普通业务量用默认的惰性清理即可。
func (s *LocalCodeStore) SetCleanupInterval(d time.Duration) {
	if d <= 0 {
		return
	}
	s.mu.Lock()
	if s.stopClean != nil {
		s.mu.Unlock()
		return // 已在清理中
	}
	s.stopClean = make(chan struct{})
	stop := s.stopClean
	s.mu.Unlock()

	go func() {
		ticker := time.NewTicker(d)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.purgeExpired()
			case <-stop:
				return
			}
		}
	}()
}

// Stop 停掉后台清理协程（启用了 SetCleanupInterval 才需要调用）
func (s *LocalCodeStore) Stop() {
	s.stopOnce.Do(func() {
		s.mu.Lock()
		stop := s.stopClean
		s.stopClean = nil
		s.mu.Unlock()
		if stop != nil {
			close(stop)
		}
	})
}

// purgeExpired 清掉所有过期条目
func (s *LocalCodeStore) purgeExpired() {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range s.codes {
		if now.After(v.expireA) {
			delete(s.codes, k)
		}
	}
	for k, expireA := range s.limit {
		if now.After(expireA) {
			delete(s.limit, k)
		}
	}
}

// Len 当前条目数（测试/排障用：看内存里积了多少）
func (s *LocalCodeStore) Len() (codes, limits int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.codes), len(s.limit)
}

// compile-time 断言
var _ CodeStore = (*LocalCodeStore)(nil)
