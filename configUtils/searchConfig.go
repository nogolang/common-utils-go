package configUtils

// SearchConfig 商品检索配置（PG GIN 标准 / 远程检索引擎标准）
type SearchConfig struct {
	// SyncMode 写侧索引同步模式：db=本地 PG GIN（默认，同一事务）；remote=mq_outbox 最终一致
	SyncMode string `json:"syncMode"`
	// RemoteUrl 读侧远程检索引擎地址（remote 模式下 RemoteSearchQueryer 使用）
	RemoteUrl string `json:"remoteUrl"`
}
