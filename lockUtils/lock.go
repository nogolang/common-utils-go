package lockUtils

import (
	"context"

	"github.com/go-redsync/redsync/v4"
)

// Locker 锁接口，支持 Redis 分布式锁和本地锁两种实现
type Locker interface {
	WithLock(ctx context.Context, key string, fn func() error) error
	Lock(ctx context.Context, key string) error
	Unlock(ctx context.Context, key string) error
}

// NewLocker 根据配置创建不同的锁实现（Redis 或 Local）
func NewLocker(cfg *LockConfig, redSync *redsync.Redsync) Locker {
	if cfg != nil {
		if cfg.Use == LockUse_Local {
			return NewLocalLock()
		} else if cfg.Use == LockUse_Redis {
			return NewRedisBizLock(redSync)
		}
	}
	panic("必须配置锁的类型")
}
