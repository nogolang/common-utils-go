package kratosUtils

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/nogolang/kratos-traefik-etcd/etcdUtils"
	"github.com/pkg/errors"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

func RegisterToEtcd(appInfo kratos.AppInfo, etcdClient *clientv3.Client, serverName string, nodeNum int, logger *zap.Logger) error {
	//让traefik识别到
	traefikEtcd := etcdUtils.NewEtcdTraefik(etcdClient,
		serverName,
		nodeNum,
	)
	err := traefikEtcd.RegisterTraefik(appInfo)
	if err != nil {
		return errors.WithMessage(err, "注册到traefik里失败")
	}
	return nil
}
