package configUtils

type AllConfig struct {
	Mode string `json:"mode"`

	Cluster struct {
		Node int `json:"node"`
	}

	SnowId struct {
		Node int `json:"node"`
	}

	//服务配置
	Server struct {
		ServerName string `json:"serverName"`
		HttpPort   int    `json:"httpPort"`
		GrpcPort   int    `json:"grpcPort"`
	}

	//日志配置
	Log struct {
		Level string `json:"level"`
	}

	//数据库配置
	Gorm struct {
		Url                  string `json:"url"`
		LogLevel             string `json:"logLevel"`
		SlowSqlMillSecond    int    `json:"slowSqlMillSecond"`
		AutoCreateForeignKey bool   `json:"autoCreateForeignKey"`
		SingularTable        bool   `json:"singularTable"`
		MaxOpenConn          int    `json:"maxOpenConn"`
	}

	//redis配置
	Redis struct {
		IsSingle   bool     `json:"isSingle"`
		SingleUrl  string   `json:"singleUrl"`
		ClusterUrl []string `json:"ClusterUrl"`
	}

	//etcd配置
	Etcd struct {
		Url []string `json:"url"`
	}
	//consul配置
	Consul struct {
		Url string `json:"url"`
	}

	Dtm struct {
		DtmServer string `json:"dtmServer"`
	}

	Jwt struct {
		Secret  string `json:"secret"`
		Role    string `json:"role"`
		Expired int    `json:"expired"` //second
	}
	Elastic struct {
		Url []string `json:"url"`
	}
	RabbitMq struct {
		Url string `json:"url"`
	}
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
