package kratosClient

import (
	"context"

	"time"

	kratosEtcd "github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/selector"
	"github.com/go-kratos/kratos/v2/selector/filter"
	"github.com/go-kratos/kratos/v2/selector/random"
	kratosHttp "github.com/go-kratos/kratos/v2/transport/http"
	"go.uber.org/zap"
)

func NewDiscoveryHttpClient(logger *zap.Logger,
	KratosEtcdClient *kratosEtcd.Registry,
	serviceName string) *kratosHttp.Client {
	//创建全局的负载均衡算法为random
	//还有p2c,wrr，具体看官方和资料
	//由于 gRPC 框架的限制，只能使用全局 balancer name 的方式来注入 selector
	selector.SetGlobalSelector(random.NewBuilder())

	//创建路由 Filter：筛选版本号为"xxx"的实例
	//这里我注册的时候就填的空,所以也为空
	filterVersion := filter.Version("")

	httpClient, err := kratosHttp.NewClient(
		context.Background(),

		//服务发现语法
		//<schema>://[authority]/<service-name>
		kratosHttp.WithEndpoint("discovery:///"+serviceName),
		kratosHttp.WithDiscovery(KratosEtcdClient),
		kratosHttp.WithNodeFilter(filterVersion),
		//WithBlock代表阻塞，如果一直没有获取到服务，则会一直阻塞，这是http独有的
		//kratosHttp.WithBlock(),

		//60秒的的超时时间
		kratosHttp.WithTimeout(time.Second*60),
	)
	if err != nil {
		logger.Fatal("服务发现初始化错误", zap.Error(err))
		return nil
	}
	return httpClient
}
