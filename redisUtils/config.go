package redisUtils

// RedisConfig redis 连接配置（单机/集群二选一，由 Single 决定）
type RedisConfig struct {
	Single bool   `json:"single"`
	//单机地址，形如 localhost:6379
	SingleUrl string `json:"singleUrl"`
	//集群地址列表（single=false 时必填）
	//⚠️ 历史坑：这里的 tag 曾写成 "ClusterUrl"（首字母大写），viper 的键一律小写，
	//导致集群模式配置永远读不到。2026-09-26 修正为全小写。
	ClusterUrl []string `json:"clusterUrl"`
	Db         int      `json:"db"`
	Username   string   `json:"username"`
	Password   string   `json:"password"`
}
