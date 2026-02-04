package configUtils

type AllConfig struct {
	Mode string `json:"mode"`

	//服务配置
	Server *ServerConfig

	SnowId *SnowIdConfig

	//日志配置
	Log *LogConfig

	//数据库配置
	Gorm *GormConfig

	//redis配置
	Redis *RedisConfig

	//etcd配置
	Etcd *EtcdConfig

	//consul配置
	Consul *ConsulConfig

	Jwt *JwtConfig

	Elastic *ElasticConfig

	RabbitMq *RabbitMqConfig
}

type ServerConfig struct {
	ServerName string `json:"serverName"`
	HttpPort   int    `json:"httpPort"`
	GrpcPort   int    `json:"grpcPort"`
}

type LogConfig struct {
	Level string `json:"level"`
}
type SnowIdConfig struct {
	Keys []string `json:"keys"`
}

type GormConfig struct {
	Url                         string `json:"url"`
	LogLevel                    string `json:"logLevel"`
	SlowSqlMillSecond           int    `json:"slowSqlMillSecond"`
	DisableAutoCreateForeignKey bool   `json:"disableAutoCreateForeignKey"`
	SingularTable               bool   `json:"singularTable"`
	MaxOpenConn                 int    `json:"maxOpenConn"`
	//是否翻译错误，比如主键冲突，你想用gorm的DUPLICATE KEY去检查是不行的，必须要先翻译
	TransError bool `json:"transError"`
}
type RedisConfig struct {
	IsSingle   bool     `json:"isSingle"`
	SingleUrl  string   `json:"singleUrl"`
	ClusterUrl []string `json:"ClusterUrl"`
}

type EtcdConfig struct {
	EnableTls bool     `json:"enableTls"`
	CaCrt     string   `json:"caCrt"`
	ClientKey string   `json:"clientKey"`
	ClientCrt string   `json:"clientCrt"`
	Url       []string `json:"url"`
}
type ConsulConfig struct {
	Url string `json:"url"`
}

type JwtConfig struct {
	Secret  string `json:"secret"`
	Role    string `json:"role"`
	Expired int    `json:"expired"` //second
}
type ElasticConfig struct {
	Url []string `json:"url"`
}
type RabbitMqConfig struct {
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
