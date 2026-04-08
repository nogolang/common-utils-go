package elasticUtils

type ElasticConfig struct {
	CaCrt     string   `json:"caCrt"`
	EnableTls bool     `json:"enableTls"`
	Username  string   `json:"username"`
	Password  string   `json:"password"`
	Url       []string `json:"url"`
}
