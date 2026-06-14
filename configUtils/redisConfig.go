package configUtils

type RedisConfig struct {
	Single     bool     `json:"single"`
	SingleUrl  string   `json:"singleUrl"`
	ClusterUrl []string `json:"ClusterUrl"`
	Db         int      `json:"db"`
}
