package etcdUtils

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	etcdClientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"go.uber.org/zap"
)

type EtcdUtils struct {
	client      *etcdClientv3.Client
	logger      *zap.Logger
	closeListen chan struct{}
	//纯洁的key，没有加任何前缀和后缀，比如userId,orderId
	pureKey  string
	wait     sync.WaitGroup
	leaseAll *sync.Map
}

func NewEtcdUtils(client *etcdClientv3.Client, logger *zap.Logger) *EtcdUtils {
	return &EtcdUtils{
		client:      client,
		logger:      logger,
		closeListen: make(chan struct{}),
		leaseAll:    new(sync.Map),
	}
}

func (receiver *EtcdUtils) GetAllKeysByPrefix(prefix string) ([]string, error) {
	res, err := receiver.client.Get(context.TODO(), prefix, etcdClientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	var keys []string
	for _, v := range res.Kvs {
		keys = append(keys, string(v.Key))
	}
	return keys, nil
}

func (receiver *EtcdUtils) GetLeaseIdByKey(key string) (*etcdClientv3.LeaseID, error) {
	value, ok := receiver.leaseAll.Load(key)
	if !ok {
		return nil, errors.New("没有找到对应的租约")
	}
	return value.(*etcdClientv3.LeaseID), nil
}

func (receiver *EtcdUtils) CreateKvWithLease(key string, value string, ttlSecond int64) error {
	//创建一个租约
	lease, keepLive, err := receiver.CreateLease(ttlSecond)
	if err != nil {
		return err
	}

	//存储租约，因为到时候会自动重新注册，所以应该用map，而不是用切片
	receiver.leaseAll.Store(receiver.pureKey, lease)

	//给租约续期
	go func() {
		err := receiver.ListenLease(keepLive, *lease, key, value, ttlSecond)
		if err != nil {
			receiver.logger.Error("重新创建租约失败", zap.Error(err))
		}
	}()

	//把key添加进去
	err = receiver.PutKey(key, value, lease)
	if err != nil {
		return err
	}

	return nil
}

// CreateUniqueNum 创建唯一序号
func (receiver *EtcdUtils) CreateUniqueNum(allNodes []int32, key string, ttlSecond int64) (int32, error) {
	prefix := "/unique/"
	receiver.pureKey = key

	//创建一个分布式锁
	session, _ := concurrency.NewSession(receiver.client,
		concurrency.WithTTL(int(ttlSecond)),
	)
	defer session.Close()
	locker := concurrency.NewMutex(session, prefix+"lock/"+key)
	ctx := context.TODO()
	err := locker.Lock(ctx)
	if err != nil {
		return 0, errors.New("系统繁忙，请稍后再试")
	}
	defer func() {
		err := locker.Unlock(ctx)
		if err != nil {
			fmt.Printf("释放锁失败%d\n", err)
			return
		}
	}()

	//获取所有key
	allKeysRaw, err := receiver.GetAllKeysByPrefix(prefix + key)
	if err != nil {
		return 0, err
	}
	var allKeys []int32
	for _, v := range allKeysRaw {
		lastIndex := strings.LastIndex(v, "/")
		atoi, err := strconv.Atoi(v[lastIndex+1:])
		if err != nil {
			return 0, errors.Wrap(err, "atoi失败")
		}
		allKeys = append(allKeys, int32(atoi))
	}

	//和allNodes做交集运算
	//获取剩下的key
	remain, _ := lo.Difference(allNodes, allKeys)

	//排序
	slices.Sort(remain)

	//取最小的num
	nowNum := remain[0]

	finalPath := prefix + key + "/" + fmt.Sprintf("%d", nowNum)

	//然后注册到etcd里
	err = receiver.CreateKvWithLease(finalPath, fmt.Sprintf("%d", nowNum), ttlSecond)
	if err != nil {
		return 0, err
	}
	return nowNum, nil
}

func (receiver *EtcdUtils) PutKey(key string, value string, lease *etcdClientv3.LeaseID) error {
	_, err := receiver.client.Put(context.TODO(), key, value, etcdClientv3.WithLease(*lease))
	if err != nil {
		return errors.Wrap(err, "put key error")
	}
	return nil
}

// 创建租约
func (receiver *EtcdUtils) CreateLease(ttlSecond int64) (*etcdClientv3.LeaseID, <-chan *etcdClientv3.LeaseKeepAliveResponse, error) {
	//创建lease对象
	lease := etcdClientv3.NewLease(receiver.client)

	//分配租约
	ctx, _ := context.WithCancel(context.Background())
	GrantRes, err := lease.Grant(ctx, ttlSecond)
	if err != nil {
		return nil, nil, err
	}

	//创建keepAlive对象，该对象可以保持续租，返回的是一个chan
	//所以我们的程序如果关闭了，这个keepAliveChain就没人取了，etcd就不会再续租了
	keepAlive, err := lease.KeepAlive(ctx, GrantRes.ID)
	if err != nil {
		return nil, nil, err
	}
	return &GrantRes.ID, keepAlive, nil
}

// 持续续租
func (receiver *EtcdUtils) ListenLease(leaseChan <-chan *etcdClientv3.LeaseKeepAliveResponse, leaseId etcdClientv3.LeaseID, key string, value string, ttlSecond int64) error {
	receiver.wait.Add(1)
	defer func() {
		receiver.wait.Done()
		//尝试释放租约，无论成功与否
		//这里主要是尝试释放之前创建的租约
		receiver.client.Revoke(context.TODO(), leaseId)
	}()

	for {
		select {
		//leaseChan会阻塞，如果是15s的TTL，那么到了5s的时候会续租一次
		//  如果是5s的TTL，那么2s就会续租一次
		case v, _ := <-leaseChan:
			//如果为nil，代表程序可能卡住了，在调试，那么就重新注册
			//  然后结束当前协程
			//当然，也有可能在平滑退出
			//  如果在平滑退出的过程中手动调用当前方法的close
			//  此时代表是我们主动关闭的，我们会等待所有协程先结束
			//  然后再去清理掉租约
			if v == nil {
				// 不是主动主动退出的，才尝试重建租约和key
				// 如果是主动退出的，那么就不会重新创建租约
				// 并且使用isClosing标志位再去判断一次
				err := receiver.CreateKvWithLease(key, value, ttlSecond)
				if err != nil {
					return errors.Wrapf(err, "租约[%d]重建失败", leaseId)
				}
				return nil
			}
		case <-receiver.closeListen:
			//收到主动关闭通知，直接退出
			return nil
		}

		time.Sleep(time.Second)
	}
}

func (receiver *EtcdUtils) RemoveLease(leaseId etcdClientv3.LeaseID) error {
	_, err := receiver.client.Revoke(context.TODO(), leaseId)
	if err != nil {
		return errors.Wrap(err, "释放租约失败")
	}
	return err
}

// 优雅关闭，这是在平滑退出的地方调用的
func (receiver *EtcdUtils) Close() {
	//关闭通知通道，通知所有续租协程退出
	close(receiver.closeListen)

	//等待所有协程结束
	receiver.wait.Wait()

	//删除所有保存的租约，因为我们已经在ListenLease里删除了，所以这里其实不需要删除
	//var allLease []*etcdClientv3.LeaseID
	//receiver.leaseAll.Range(func(key, value interface{}) bool {
	//	allLease = append(allLease, value.(*etcdClientv3.LeaseID))
	//	return true
	//})
	//for _, v := range allLease {
	//	receiver.RemoveLease(*v)
	//}
}
