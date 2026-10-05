package elasticUtils

// ElasticConfig elasticsearch 连接配置
type ElasticConfig struct {
	CaCrt     string   `json:"caCrt"`
	EnableTls bool     `json:"enableTls"`
	Username  string   `json:"username"`
	Password  string   `json:"password"`
	Url       []string `json:"url"`
}
