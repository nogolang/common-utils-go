package dtmUtils

import (
	"log/slog"
	"strconv"
	"strings"

	"github.com/dtm-labs/dtm/client/dtmcli"
	rawMysql "github.com/go-sql-driver/mysql"
)

// NewDtmDbConfig 由数据库 DSN 推出 dtm 的 DBConf。
//
// 只收 DSN 字符串而不是整份 gorm 配置：dtm 只需要连接信息，
// 让它依赖 gormUtils 会把两个不相干的工具包绑在一起。
func NewDtmDbConfig(dsn string) *dtmcli.DBConf {
	cfg, err := rawMysql.ParseDSN(dsn)
	if err != nil {
		slog.Error("解析数据库连接字符串出错", "err", err)
		return nil
	}
	// 在xa模式下，每个数据库实例都是一个rm，然后最终交给dtm管理
	index := strings.LastIndex(cfg.Addr, ":")
	if index < 0 {
		slog.Error("数据库地址错误", "addr", cfg.Addr)
		return nil
	}
	host := cfg.Addr[:index]
	port, err := strconv.ParseInt(cfg.Addr[index+1:], 10, 64)
	if err != nil {
		slog.Error("数据库端口错误", "err", err)
		return nil
	}
	obj := &dtmcli.DBConf{
		Driver:   "mysql",
		Host:     host,
		Port:     port,
		User:     cfg.User,
		Password: cfg.Passwd,
		Db:       cfg.DBName,
	}
	return obj
}
