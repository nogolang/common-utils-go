package bizLock

import (
	"context"
	"fmt"

	"github.com/go-redsync/redsync/v4"
	"github.com/pkg/errors"
)

// RedisLock redsync 实现的分布式锁
type RedisLock struct {
	rs *redsync.Redsync
}

func NewRedisBizLock(rs *redsync.Redsync) *RedisLock {
	return &RedisLock{
		rs: rs,
	}
}

func (r *RedisLock) Lock(ctx context.Context, str string) error {
	mutex := r.rs.NewMutex(str)
	err := mutex.LockContext(ctx)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("获取锁失败%s", str))
	}
	return nil
}

func (r *RedisLock) Unlock(ctx context.Context, str string) error {
	mutex := r.rs.NewMutex(str)
	unlocked, err := mutex.UnlockContext(ctx)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("释放锁失败%s", str))
	}
	if !unlocked {
		return errors.New(fmt.Sprintf("锁不存在或不是当前持有者%s", str))
	}
	return nil
}
