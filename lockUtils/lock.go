package lockUtils

import (
	"context"

	"github.com/go-redsync/redsync/v4"
	"github.com/nogolang/common-utils-go/configUtils"
)

type Locker interface {
	Lock(ctx context.Context, id string) error
	Unlock(ctx context.Context, id string) error
}

// 根据配置创建不同的的锁
func NewLocker(allConfig *configUtils.CommonConfig, redSync *redsync.Redsync) Locker {
	if allConfig.Lock.Use == configUtils.LockUse_Local {
		return NewLocalLock()
	} else if allConfig.Lock.Use == configUtils.LockUse_Redis {
		return NewRedisBizLock(redSync)
	}
	panic("必须配置锁的类型")
	return nil
}
