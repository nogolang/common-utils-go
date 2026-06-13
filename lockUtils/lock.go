package lockUtils

import (
	"context"
	"strings"

	"github.com/go-redsync/redsync/v4"
	"github.com/nogolang/common-utils-go/configUtils"
)

// Locker 锁接口，支持 Redis 分布式锁和本地锁两种实现
type Locker interface {
	Lock(ctx context.Context, id string) error
	Unlock(ctx context.Context, id string) error
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

// WithLock 在锁保护下执行函数，自动加锁和释放锁。
// 如果锁在业务执行期间已过期（看门狗未能续期），释放时仅记录警告而非错误。
func WithLock(ctx context.Context, locker Locker, id string, fn func() error) error {
	if err := locker.Lock(ctx, id); err != nil {
		return err
	}
	defer func() {
		if err := locker.Unlock(ctx, id); err != nil {
			// 仅当锁已过期时忽略错误（业务已完成，锁超时是预期情况）
			if strings.Contains(err.Error(), "lock was already expired") ||
				strings.Contains(err.Error(), "锁过期") {
				return
			}
		}
	}()
	return fn()
}
