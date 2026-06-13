package lockUtils

import (
	"context"
	"fmt"
	"sync"

	"github.com/pkg/errors"
)

// LocalLock 本地锁实现，基于 sync.Mutex，仅适用于单实例场景
type LocalLock struct {
	lockMap sync.Map // map[string]*sync.Mutex
}

func NewLocalLock() *LocalLock {
	return &LocalLock{
		lockMap: sync.Map{},
	}
}

func (l *LocalLock) Lock(_ context.Context, id string) error {
	mu := l.getMutex(id)
	mu.Lock()
	return nil
}

func (l *LocalLock) Unlock(_ context.Context, id string) error {
	mu, ok := l.lockMap.Load(id)
	if !ok {
		return errors.New(fmt.Sprintf("锁不存在：%s", id))
	}
	realMu, ok := mu.(*sync.Mutex)
	if !ok {
		return errors.New(fmt.Sprintf("锁类型错误：%s", id))
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
