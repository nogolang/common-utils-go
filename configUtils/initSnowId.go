package configUtils

import (
	"log"

	"github.com/bwmarrin/snowflake"

	EtcdClientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

// 获取到snow对象，需要手动指定node
// 每个服务的每个业务都可以指定1024个，所以肯定够用的
func NewSnowflake(client *EtcdClientv3.Client,
	AllConfig *AllConfig) *snowflake.Node {
	newNode, err := snowflake.NewNode(int64(AllConfig.SnowId.Node))
	if err != nil {
		log.Fatal("创建snowNode失败", zap.Error(err))
		return nil
	}
	return newNode
}
