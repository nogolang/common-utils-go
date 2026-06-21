package lockUtils

import (
	"context"

	"github.com/go-redsync/redsync/v4"
	"github.com/nogolang/common-utils-go/configUtils"
)

// Locker 锁接口，支持 Redis 分布式锁和本地锁两种实现
type Locker interface {
	WithLock(ctx context.Context, key string, fn func() error) error
	Lock(ctx context.Context, key string) error
	Unlock(ctx context.Context, key string) error
}

// NewLocker 根据配置创建不同的锁实现（Redis 或 Local）
func NewLocker(allConfig *configUtils.CommonConfig, redSync *redsync.Redsync) Locker {
	if allConfig.Lock.Use == configUtils.LockUse_Local {
		return NewLocalLock()
	} else if allConfig.Lock.Use == configUtils.LockUse_Redis {
		return NewRedisBizLock(redSync)
	}
	panic("必须配置锁的类型")
}
