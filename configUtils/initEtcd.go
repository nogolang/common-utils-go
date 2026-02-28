package configUtils

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"log"
	"os"
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
func NewKratosEtcdClient(etcdClient *clientv3.Client, logger *zap.Logger) *etcd.Registry {
	r := etcd.New(etcdClient,
		//注册到etcd中的租约TTL
		etcd.RegisterTTL(time.Second*15),
	)
	logger.Sugar().Info("连接etcd成功")

	selector.SetGlobalSelector(random.NewBuilder())

	//注册全局的resolver
	//  这样dtm就可以直接使用资源服务的服务名称去调用了，而不用指定资源服务的的服务地址
	//  并且dtm自身的地址也可以通过discovery:///dtmservice来获取（前提我们在dtm启动的时配置了）
	//  这样就可以用dtm集群了
	resolver.Register(discovery.NewBuilder(r, discovery.WithInsecure(true)))
	return r
}

func NewEtcdClient(allConfig *CommonConfig, logger *zap.Logger) *clientv3.Client {
	var crt tls.Config
	var etcdConfig clientv3.Config
	if allConfig.Etcd.EnableTls {
		caCrtData, err := os.ReadFile(allConfig.Etcd.CaCrt)
		if err != nil {
			logger.Sugar().Fatal("读取etcd CA根证书失败: ", err.Error())
		}
		// 初始化证书池，nil表示基于系统根证书池，若仅信任自定义CA则用x509.NewCertPool()
		certPool := x509.NewCertPool()
		// 将CA证书添加到证书池，解析失败会返回false
		if !certPool.AppendCertsFromPEM(caCrtData) {
			logger.Sugar().Fatal("解析etcd CA根证书失败，证书格式错误")
		}

		clientCert, err := tls.LoadX509KeyPair(allConfig.Etcd.ClientCrt, allConfig.Etcd.ClientKey)
		if err != nil {
			logger.Sugar().Fatal("加载客户端证书/私钥对失败: ", err.Error())
		}
		crt = tls.Config{
			ServerName:   "etcd",
			RootCAs:      certPool,
			Certificates: []tls.Certificate{clientCert},
		}
		//创建etcd配置
		etcdConfig = clientv3.Config{
			Endpoints: allConfig.Etcd.Url,
			TLS:       &crt,
		}
	} else {
		etcdConfig = clientv3.Config{
			Endpoints: allConfig.Etcd.Url,
		}
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
