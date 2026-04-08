package gormUtils

type GormConfig struct {
	NoUrl                       bool   `json:"noUrl"`
	Url                         string `json:"url"`
	Username                    string `json:"username"`
	Password                    string `json:"password"`
	Host                        string `json:"host"`
	Database                    string `json:"database"`
	Param                       string `json:"param"`
	LogLevel                    string `json:"logLevel"`
	SlowSqlMillSecond           int    `json:"slowSqlMillSecond"`
	DisableAutoCreateForeignKey bool   `json:"disableAutoCreateForeignKey"`
	SingularTable               bool   `json:"singularTable"`
	MaxOpenConn                 int    `json:"maxOpenConn"`
	//是否翻译错误，比如主键冲突，你想用gorm的DUPLICATE KEY去检查是不行的，必须要先翻译
	TransError bool `json:"transError"`
}
