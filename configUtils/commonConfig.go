package configUtils

import "os"

type CommonConfig struct {
	//服务配置
	Server *serverConfig

	//当需要从etcd里获取唯一数字的时候，才需要配置
	//  但是一般不会用，k8s更好
	SnowId *snowIdConfig

	//公共配置路径
	CommonConfigPath []string `json:"commonConfigPath"`

	//日志配置
	Log *logConfig

	//数据库配置
	Gorm *gormConfig

	//redis配置
	Redis *redisConfig

	//etcd配置
	Etcd *etcdConfig

	//consul配置
	Consul *consulConfig

	Jwt *jwtConfig

	Elastic *elasticConfig

	RabbitMq *rabbitMqConfig

	Upload *uploadConfig

	//dtm的配置，主要是配置日志之类的
	Dtm *dtmConfig
}

type dtmConfig struct {
	LogLevel string `json:"logLevel"`
}

type serverConfig struct {
	ServerName string `json:"serverName"`
	HttpPort   int    `json:"httpPort"`
	GrpcPort   int    `json:"grpcPort"`
}

type logConfig struct {
	Level string `json:"level"`
}
type snowIdConfig struct {
	Keys []string `json:"keys"`
}

type uploadConfig struct {
	NowUse    string     `json:"nowUse"`
	AliYunOss *aliYunOss `json:"aliYunOss"`

	//下面是form签名的限制条件
	IncludeType   []string `json:"includeType"`
	MinUploadSize string   `json:"minUploadSize"`
	MaxUploadSize string   `json:"maxUploadSize"`
}

type aliYunOss struct {
	AccessKeyId     string `json:"accessKeyId"`
	AccessKeySecret string `json:"accessKeySecret"`
	BucketName      string `json:"bucketName"`
	Endpoint        string `json:"endpoint"`
	Region          string `json:"region"`
}

type gormConfig struct {
	NoUrl                       bool              `json:"noUrl"`
	Url                         string            `json:"url"`
	Username                    string            `json:"username"`
	Password                    string            `json:"password"`
	Host                        string            `json:"host"`
	Database                    string            `json:"database"`
	Param                       map[string]string `json:"param"`
	LogLevel                    string            `json:"logLevel"`
	SlowSqlMillSecond           int               `json:"slowSqlMillSecond"`
	DisableAutoCreateForeignKey bool              `json:"disableAutoCreateForeignKey"`
	SingularTable               bool              `json:"singularTable"`
	MaxOpenConn                 int               `json:"maxOpenConn"`
	//是否翻译错误，比如主键冲突，你想用gorm的DUPLICATE KEY去检查是不行的，必须要先翻译
	TransError bool `json:"transError"`
}
type redisConfig struct {
	IsSingle   bool     `json:"isSingle"`
	SingleUrl  string   `json:"singleUrl"`
	ClusterUrl []string `json:"ClusterUrl"`
}

type etcdConfig struct {
	EnableTls bool     `json:"enableTls"`
	CaCrt     string   `json:"caCrt"`
	ClientKey string   `json:"clientKey"`
	ClientCrt string   `json:"clientCrt"`
	Url       []string `json:"url"`
}
type consulConfig struct {
	Url string `json:"url"`
}

type jwtConfig struct {
	Secret  string `json:"secret"`
	Role    string `json:"role"`
	Expired int    `json:"expired"` //second
}
type elasticConfig struct {
	CaCrt     string   `json:"caCrt"`
	EnableTls bool     `json:"enableTls"`
	Username  string   `json:"username"`
	Password  string   `json:"password"`
	Url       []string `json:"url"`
}
type rabbitMqConfig struct {
	Url string `json:"url"`
}

// 判断是否是开发环境
func IsDev() bool {
	isProd := os.Getenv("PROD")
	if isProd != "" {
		return false
	}
	return true
}
