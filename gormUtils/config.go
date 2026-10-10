package gormUtils

// GormConfig 数据库配置
//
// 2026-09-26 配置解耦：从原 configUtils 下沉到本包。原先 NewGorm 吃的是
// 整个 CommonConfig，但它只用到 Gorm 这一段——调用方为了建一个库连接，
// 必须先凑出一份含 17 段的配置聚合体，这是纯粹的负担。
type GormConfig struct {
	DatabaseType string `json:"databaseType"`
	Url          string `json:"url"`
	LogLevel     string `json:"logLevel"`
	//慢查询阈值（毫秒）：超过即按 slow 级别打日志
	SlowSqlMillSecond int `json:"slowSqlMillSecond"`
	//是否关闭自动创建外键。
	//⚠️ 历史坑：yaml 里曾写成 autoCreateForeignKey，语义正好相反且与本 tag 对不上，
	//导致这个开关从上线起从未生效过。2026-09-26 已把 common.dev/prod.yaml 的键名改成本字段名。
	DisableAutoCreateForeignKey bool `json:"disableAutoCreateForeignKey"`
	SingularTable               bool `json:"singularTable"`
	MaxOpenConn                 int  `json:"maxOpenConn"`
	//连接最长存活时间（分钟）：到期的池连接被关闭换新——默认 5 分钟（<=0 回落 5）。
	//为什么需要：zhparser 表级词典（sync_zhprs_custom_word）只对"新连接"生效，
	//池里长生不死的旧连接会一直用旧词典；给个生命周期让词典更新在窗口内扩散到全部连接
	ConnMaxLifetimeMinutes int `json:"connMaxLifetimeMinutes"`
	//是否翻译驱动错误，比如主键冲突你想用 gorm 的 ErrDuplicatedKey 判断就必须先翻译
	TransError bool `json:"transError"`
}
