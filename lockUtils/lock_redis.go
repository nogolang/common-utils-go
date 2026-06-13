package lockUtils

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/pkg/errors"
)

// RedisLock 基于 redsync 实现的分布式锁，带有看门狗自动续期机制。
//
// 看门狗工作原理：
//   - redsync 默认锁 TTL 为 8 秒
//   - 看门狗每 4 秒（TTL 的一半）调用 Extend 续期一次
//   - 看门狗在以下情况退出：Unlock 信号、context 取消、续期失败
type RedisLock struct {
	rs       *redsync.Redsync
	mutexMap sync.Map // map[string]*redsync.Mutex  存储已获取的锁对象
	doneMap  sync.Map // map[string]chan struct{}    看门狗终止信号
	prefix   string
}

// NewRedisBizLock 创建 Redis 分布式锁实例
func NewRedisBizLock(rs *redsync.Redsync) *RedisLock {
	return &RedisLock{
		rs:     rs,
		prefix: "bizLock:",
	}
}

// Lock 获取分布式锁，并启动看门狗协程自动续期
func (r *RedisLock) Lock(ctx context.Context, id string) error {
	key := r.prefix + id
	mutex := r.rs.NewMutex(key) // redsync 默认: 8s TTL, 32 次重试, 512ms 重试间隔

	if err := mutex.LockContext(ctx); err != nil {
		return errors.Wrap(err, fmt.Sprintf("获取锁失败: %s", id))
	}

	// 保存 mutex 对象，Unlock 时需要使用同一个实例（内含唯一 token）
	r.mutexMap.Store(id, mutex)

	// 创建看门狗终止信号通道
	done := make(chan struct{})
	r.doneMap.Store(id, done)

	// 启动看门狗协程：定时续期，防止业务执行时间超过锁 TTL
	go r.watchdog(mutex, done, id, ctx)

	return nil
}

// watchdog 看门狗协程，定时续期锁，防止锁在业务执行期间过期
func (r *RedisLock) watchdog(mutex *redsync.Mutex, done <-chan struct{}, id string, ctx context.Context) {
	// 续期间隔为 TTL 的一半（4秒），确保在锁过期前完成续期
	ticker := time.NewTicker(time.Second * 4)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			ok, err := mutex.Extend()
			if err != nil || !ok {
				// 续期失败，看门狗退出。
				// 可能原因：Redis 不可达、锁已被其他客户端获取等
				return
			}
		case <-done:
			// Unlock 信号，正常退出
			return
		case <-ctx.Done():
			// 上下文取消，退出
			return
		}
	}
}

// Unlock 释放分布式锁，并停止看门狗协程
func (r *RedisLock) Unlock(ctx context.Context, id string) error {
	// 1. 取出并删除保存的 mutex（原子操作，防止重复释放）
	val, ok := r.mutexMap.LoadAndDelete(id)
	if !ok {
		return errors.New(fmt.Sprintf("锁不存在: %s", id))
	}
	mutex := val.(*redsync.Mutex)

	// 2. 通知看门狗停止续期
	if doneVal, ok := r.doneMap.LoadAndDelete(id); ok {
		close(doneVal.(chan struct{}))
	}

	// 3. 释放 Redis 锁（使用同一个 mutex 实例，保证 token 匹配）
	unlocked, err := mutex.UnlockContext(ctx)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("释放锁失败: %s", id))
	}
	if !unlocked {
		return errors.New(fmt.Sprintf("锁不存在或不是当前持有者: %s", id))
	}

	return nil
}
