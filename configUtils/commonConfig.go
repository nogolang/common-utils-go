package configUtils

import (
	"os"

	"github.com/nogolang/common-utils-go/cloudUtils"
	"github.com/nogolang/common-utils-go/consulUtils"
	"github.com/nogolang/common-utils-go/dtmUtils"
	"github.com/nogolang/common-utils-go/elasticUtils"
	"github.com/nogolang/common-utils-go/etcdUtils"
	"github.com/nogolang/common-utils-go/gormUtils"
	"github.com/nogolang/common-utils-go/redisUtils"
	"github.com/nogolang/common-utils-go/tokenUtils"
	"github.com/nogolang/common-utils-go/uploadUtils"
	"github.com/nogolang/common-utils-go/watermillUtils"
)

type CommonConfig struct {
	//服务配置
	Server *serverConfig

	//公共配置路径
	CommonConfigPath []string `json:"commonConfigPath"`

	//日志配置
	Log *logConfig

	//数据库配置
	Gorm *gormUtils.GormConfig

	//redis配置
	Redis *redisUtils.RedisConfig

	//etcd配置
	Etcd *etcdUtils.EtcdConfig

	//consul配置
	Consul *consulUtils.ConsulConfig

	Jwt *tokenUtils.JwtConfig

	Elastic *elasticUtils.ElasticConfig

	RabbitMq *watermillUtils.RabbitMqConfig

	//dtm的配置，主要是配置日志之类的
	Dtm *dtmUtils.DtmConfig

	//阿里云账户
	AliYunAccount *cloudUtils.AliYunAccount `json:"aliYunAccount"`
	Upload        *uploadUtils.UploadConfig `json:"upload"`
}

type serverConfig struct {
	ServerName string `json:"serverName"`
	HttpPort   int    `json:"httpPort"`
	GrpcPort   int    `json:"grpcPort"`
}

type logConfig struct {
	Level string `json:"level"`
}

// 判断是否是开发环境
func IsDev() bool {
	isProd := os.Getenv("PROD")
	if isProd != "" {
		return false
	}
	return true
}
