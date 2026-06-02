package bizLock

import (
	"context"
	"fmt"
	"sync"

	"github.com/pkg/errors"
)

// LocalLock 本地锁实现
type LocalLock struct {
	lockMap sync.Map
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

func (l *LocalLock) getMutex(id string) *sync.Mutex {
	mu, ok := l.lockMap.Load(id)
	if !ok {
		newMu := &sync.Mutex{}
		l.lockMap.Store(id, newMu)
		return newMu
	}
	return mu.(*sync.Mutex)
}
