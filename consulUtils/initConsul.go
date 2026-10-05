package consulUtils

import (
	"fmt"

	consulRegister "github.com/go-kratos/kratos/contrib/registry/consul/v2"
	"github.com/go-kratos/kratos/v2/log"
	consulApi "github.com/hashicorp/consul/api"

	"go.uber.org/zap"
)

// NewKratosConsulClient 注册到 consul，并把服务端口写进 traefik tags。
//
// httpPort/grpcPort 单独传入而不是读配置聚合体：consul 包只关心端口，
// 端口归业务所有（服务自己的 configs/*.yaml），不该由库来定义。
func NewKratosConsulClient(cfg *ConsulConfig, httpPort, grpcPort int) *consulRegister.Registry {
	client, err := consulApi.NewClient(&consulApi.Config{
		Address: cfg.Url,
	})
	if err != nil {
		log.Fatal("连接consul失败", zap.Error(err))
	}
	log.Info("连接consul成功")

	//需要把官方的kratos替换为我们自己修改支持consul的，里面支持了自定义的tags，具体看go mod
	//这里只需要配置注册的信息即可，其他配置可以由文件来配置
	tags := []string{
		"traefik.enable=true",
		//设置端口，默认kratos里的port是grpc的端口，这里我们暴露给traefik http的端口
		//如果是grpc，则需要设置traefik.http.services.service-name.loadbalancer.server.scheme=h2c
		//并且我们把grpc设置为不同的service，router也要重新提供一份
		"traefik.http.services.user-service.loadBalancer.server.port=" + fmt.Sprintf("%d", httpPort),
		"traefik.http.services.user-service-grpc.loadBalancer.server.port=" + fmt.Sprintf("%d", grpcPort),
		"traefik.http.services.user-service-grpc.loadBalancer.server.scheme=h2c",
	}

	//最新版本可以支持tags了
	registry := consulRegister.New(client, consulRegister.WithTags(tags))
	return registry
}
