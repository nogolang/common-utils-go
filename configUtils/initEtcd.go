package configUtils

import (
	"context"
	"log"
	"time"

	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/selector"
	"github.com/go-kratos/kratos/v2/selector/random"
	"github.com/go-kratos/kratos/v2/transport/grpc/resolver/discovery"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc/resolver"

	// 导入 kratos 的 dtm 驱动
	_ "github.com/dtm-labs/driver-kratos"
)

// 同时返回etcd-kratos-registry和etcd client
func NewKratosEtcdClient(etcdClient *clientv3.Client) *etcd.Registry {
	r := etcd.New(etcdClient,
		etcd.RegisterTTL(time.Second*5),
	)
	log.Println("连接etcd成功")

	selector.SetGlobalSelector(random.NewBuilder())

	//注册全局的resolver
	//  这样dtm就可以直接使用资源服务的服务名称去调用了，而不用指定资源服务的的服务地址
	//  并且dtm自身的地址也可以通过discovery:///dtmservice来获取（前提我们在dtm启动的时配置了）
	//  这样就可以用dtm集群了
	resolver.Register(
		discovery.NewBuilder(r, discovery.WithInsecure(true)))
	return r
}

func NewEtcdClient(allConfig *AllConfig) *clientv3.Client {

	//指定所有的endpoints
	etcdConfig := clientv3.Config{
		Endpoints: allConfig.Etcd.Url,
	}

	//3.3x版本以后，超时不会直接通过error返回，必须要使用Status方法判断
	client, _ := clientv3.New(etcdConfig)
	timeout, _ := context.WithTimeout(context.Background(), 3*time.Second)
	_, err := client.Status(timeout, etcdConfig.Endpoints[0])
	if err != nil {
		log.Fatal("连接etcd失败", zap.Error(err))
		return nil
	}

	return client
}
