package configUtils

import (
	"fmt"

	consulRegister "github.com/go-kratos/kratos/contrib/registry/consul/v2"
	"github.com/go-kratos/kratos/v2/log"
	consulApi "github.com/hashicorp/consul/api"

	"go.uber.org/zap"
)

func NewKratosConsulClient(allConfig *CommonConfig) *consulRegister.Registry {
	client, err := consulApi.NewClient(&consulApi.Config{
		Address: allConfig.Consul.Url,
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
		"traefik.http.services.user-service.loadBalancer.server.port=" + fmt.Sprintf("%d", allConfig.Server.HttpPort),
		"traefik.http.services.user-service-grpc.loadBalancer.server.port=" + fmt.Sprintf("%d", allConfig.Server.GrpcPort),
		"traefik.http.services.user-service-grpc.loadBalancer.server.scheme=h2c",
	}

	//最新版本可以支持tags了
	registry := consulRegister.New(client, consulRegister.WithTags(tags))
	return registry
}
