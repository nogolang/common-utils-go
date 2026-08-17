package configUtils

import (
	"os"

	"github.com/nogolang/common-utils-go/uploadUtils"
)

type CommonConfig struct {
	//服务配置
	Server *serverConfig

	//公共配置路径
	CommonConfigPath []string `json:"commonConfigPath"`

	//日志配置
	Log *LogConfig `json:"log"`

	//数据库配置
	Gorm *GormConfig

	//锁的类型
	Lock *LockConfig `json:"lock"`

	//redis配置
	Redis *RedisConfig `json:"redis"`

	//etcd配置
	Etcd *EtcdConfig `json:"etcd"`

	//consul配置
	Consul *ConsulConfig `json:"consul"`

	Jwt *JwtConfig `json:"jwt"`

	Elastic *ElasticConfig `json:"elastic"`

	RabbitMq *RabbitMqConfig `json:"rabbitMq"`

	//dtm的配置，主要是配置日志之类的
	Dtm *DtmConfig `json:"dtm"`

	//阿里云账户
	AliYunAccount *AliYunAccount            `json:"aliYunAccount"`
	Upload        *uploadUtils.UploadConfig `json:"upload"`
}

type serverConfig struct {
	ServerName string `json:"serverName"`
	HttpPort   int    `json:"httpPort"`
	GrpcPort   int    `json:"grpcPort"`
}

type LogConfig struct {
	Level string `json:"level"`
	//默认使用zap
	Use string `json:"use"`
	//隐藏的字段
	HiddenField []string `json:"hiddenField"`
}

// 判断是否是开发环境
func IsDev() bool {
	is := os.Getenv("PROJECT")
	if is == "dev" || is == "" {
		return true
	}
	return false
}
func IsProd() bool {
	is := os.Getenv("PROJECT")
	if is == "prod" {
		return true
	}
	return false
}

func IsTest() bool {
	is := os.Getenv("PROJECT")
	if is == "test" {
		return true
	}
	return false
}
