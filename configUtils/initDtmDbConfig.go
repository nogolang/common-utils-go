package configUtils

import (
	"strconv"
	"strings"

	"github.com/dtm-labs/dtm/client/dtmcli"
	rawMysql "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
)

func NewDtmDbConfig(allConfig *CommonConfig) *dtmcli.DBConf {
	cfg, err := rawMysql.ParseDSN(allConfig.Gorm.Url)
	if err != nil {
		zap.L().Error("解析数据库连接字符串出错")
		return nil
	}
	// 在xa模式下，每个数据库实例都是一个rm，然后最终交给dtm管理
	index := strings.LastIndex(cfg.Addr, ":")
	if index < 0 {
		zap.L().Error("数据库地址错误")
	}
	host := cfg.Addr[:index]
	port, err := strconv.ParseInt(cfg.Addr[index+1:], 10, 64)
	if err != nil {
		zap.L().Error("数据库端口错误")
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
