package etcdUtils

type EtcdConfig struct {
	EnableTls bool     `json:"enableTls"`
	CaCrt     string   `json:"caCrt"`
	ClientKey string   `json:"clientKey"`
	ClientCrt string   `json:"clientCrt"`
	Url       []string `json:"url"`
}
