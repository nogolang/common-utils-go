package configUtils

import (
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

	//商品检索配置（PG GIN 标准 / 远程检索引擎标准）
	Search *SearchConfig `json:"search"`

	//雪花ID配置
	Snow *SnowConfig `json:"snow"`

	//k8s部署相关配置
	K8s *K8sConfig `json:"k8s"`

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
	//编码格式：console（默认，人读） / json（生产日志采集用）
	Encoder string `json:"encoder"`
	//输出位置：console（默认，全部输出到控制台） / file（all+error 分文件落盘，控制台只输出error，方便pod里查看，避免撑爆容器volume）
	Output string `json:"output"`
}

// SnowConfig 雪花ID配置
type SnowConfig struct {
	//直接指定节点号，本地开发用；为0视为未设置
	WorkerId int64 `json:"workerId"`
	//从 POD_NAME 解析节点号（K8s StatefulSet 部署用），优先级高于 workerId
	WorkerIdFromPodName bool `json:"workerIdFromPodName"`
}

// K8sConfig k8s部署相关配置
type K8sConfig struct {
	//是否注册到 k8s 服务注册表；false 时返回空 registry，不注册（本地开发用）
	Register bool `json:"register"`
}
