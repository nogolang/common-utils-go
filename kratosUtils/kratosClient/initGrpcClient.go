package kratosClient

import (
	"context"
	"fmt"

	kratosEtcd "github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/selector"
	"github.com/go-kratos/kratos/v2/selector/filter"
	"github.com/go-kratos/kratos/v2/selector/random"
	kratosGrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/nogolang/common-utils-go/kratosUtils/kratosMiddleware"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	rawGrpc "google.golang.org/grpc"
)

func GetGrpcClient(logger *zap.Logger,
	serverName string,
	kratosEtcdClient *kratosEtcd.Registry) (*rawGrpc.ClientConn, error) {
	//创建全局的负载均衡算法为random
	//还有p2c,wrr，具体看官方和资料
	//由于 gRPC 框架的限制，只能使用全局 balancer name 的方式来注入 selector
	selector.SetGlobalSelector(random.NewBuilder())

	//创建路由 Filter：筛选版本号为"xxx"的实例
	//这里我注册的时候就填的空,所以也为空
	filterVersion := filter.Version("")

	//正常来说应该使用Dail，但是我们目前没有证书，只能使用Insecure
	grpcClient, err := kratosGrpc.DialInsecure(
		context.Background(),
		//服务发现语法
		//<schema>://[authority]/<service-name>
		kratosGrpc.WithEndpoint("discovery:///"+serverName),
		kratosGrpc.WithDiscovery(kratosEtcdClient),
		kratosGrpc.WithNodeFilter(filterVersion),
		kratosGrpc.WithMiddleware(kratosMiddleware.LoggerClientMiddleware(logger)),
	)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("获取 %s 服务错误", serverName))
	}
	return grpcClient, nil
}
