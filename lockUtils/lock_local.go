package lockUtils

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

// LocalLock 本地锁实现，基于 sync.Mutex，仅适用于单实例场景
type LocalLock struct {
	lockMap sync.Map // map[string]*sync.Mutex
}

func (l *LocalLock) WithLock(ctx context.Context, key string, fn func() error) error {
	if err := l.Lock(ctx, key); err != nil {
		return err
	}
	defer func() {
		if err := l.Unlock(ctx, key); err != nil {
			//可能是因为断点GC等问题
			//需要记录错误日志
			if strings.Contains(err.Error(), "lock was already expired") {
				slog.ErrorContext(ctx, fmt.Sprintf("释放锁失败，锁已过期，key=%s, err=%+v", key, err))
				return
			}
			slog.ErrorContext(ctx, fmt.Sprintf("释放锁失败，key=%s, err=%+v", key, err))
		}
	}()
	return fn()
}

func NewLocalLock() *LocalLock {
	return &LocalLock{
		lockMap: sync.Map{},
	}
}

func (l *LocalLock) Lock(_ context.Context, key string) error {
	mu := l.getMutex(key)
	mu.Lock()
	return nil
}

func (l *LocalLock) Unlock(_ context.Context, key string) error {
	mu, ok := l.lockMap.Load(key)
	if !ok {
		return errors.New(fmt.Sprintf("锁不存在：%s", key))
	}
	realMu, ok := mu.(*sync.Mutex)
	if !ok {
		return errors.New(fmt.Sprintf("锁类型错误：%s", key))
	}
	realMu.Unlock()
	return nil
}

// getMutex 获取或创建指定 id 的互斥锁。
// 使用 LoadOrStore 保证原子性，避免并发创建时出现竞态条件。
func (l *LocalLock) getMutex(id string) *sync.Mutex {
	newMu := &sync.Mutex{}
	mu, _ := l.lockMap.LoadOrStore(id, newMu)
	return mu.(*sync.Mutex)
}
