package configUtils

type AllConfig struct {
	Mode string `json:"mode"`

	//服务配置
	Server *serverConfig

	SnowId *snowIdConfig

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

type gormConfig struct {
	Url                         string `json:"url"`
	LogLevel                    string `json:"logLevel"`
	SlowSqlMillSecond           int    `json:"slowSqlMillSecond"`
	DisableAutoCreateForeignKey bool   `json:"disableAutoCreateForeignKey"`
	SingularTable               bool   `json:"singularTable"`
	MaxOpenConn                 int    `json:"maxOpenConn"`
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
	Url []string `json:"url"`
}
type rabbitMqConfig struct {
	Url string `json:"url"`
}

// 判断开发还是生产环境
func (receiver *AllConfig) IsDev() bool {
	if receiver.Mode == "dev" {
		return true
	} else if receiver.Mode == "prod" {
		return false
	} else if receiver.Mode == "" {
		return true
	}
	return true
}
