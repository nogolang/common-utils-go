package configUtils

import (
	"net/url"
	"time"

	mysqlUtil "github.com/go-sql-driver/mysql"
	"github.com/nogolang/gorm-zap/gormZap"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// 初始化gorm的配置
func getGormConfigCommon(logger *zap.Logger, allConfig *CommonConfig) *gorm.Config {
	var gormLogLevel gormlogger.LogLevel
	switch allConfig.Gorm.LogLevel {
	case "info":
		gormLogLevel = gormlogger.Info
	case "warn":
		gormLogLevel = gormlogger.Warn
	case "error":
		gormLogLevel = gormlogger.Error
	default:
		gormLogLevel = gormlogger.Info
	}
	var config = &gorm.Config{
		Logger: gormZap.NewGormZap(logger, gormLogLevel,
			time.Duration(allConfig.Gorm.SlowSqlMillSecond)*time.Millisecond), //gorm适配zap
		NamingStrategy: schema.NamingStrategy{
			SingularTable: allConfig.Gorm.SingularTable,
		},
		//直接写死
		TranslateError: true,
		//是否关闭自动创建外键
		DisableForeignKeyConstraintWhenMigrating: allConfig.Gorm.DisableAutoCreateForeignKey,
	}
	return config
}

func SetGormThread(db *gorm.DB, allConfig *CommonConfig) error {
	raw, err := db.DB()
	if err != nil {
		return err
	}

	//设置最大连接数，需要同时设置数据库本身
	raw.SetMaxOpenConns(allConfig.Gorm.MaxOpenConn)
	return nil
}

// NewGormConfig logger由外部注入进来
func NewGorm(logger *zap.Logger, allConfig *CommonConfig) *gorm.DB {
	config := getGormConfigCommon(logger, allConfig)
	finalDns := ""
	//如果不显示的配置不使用url，那默认就是使用url
	if allConfig.Gorm.NoUrl {
		cfg := mysqlUtil.NewConfig()
		cfg.DBName = allConfig.Gorm.Database
		cfg.User = allConfig.Gorm.Username
		cfg.Passwd = allConfig.Gorm.Password
		cfg.Addr = allConfig.Gorm.Host
		urlValues, err := url.ParseQuery(allConfig.Gorm.Param)
		if err != nil {
			logger.Fatal("url解码失败", zap.Error(err))
			return nil
		}
		dbParam := make(map[string]string)
		for k, v := range urlValues {
			dbParam[k] = v[0]
		}
		cfg.Params = dbParam
		cfg.Net = "tcp"
		finalDns = cfg.FormatDSN()
	} else {
		finalDns = allConfig.Gorm.Url
	}

	logger.Info("要连接的数据库", zap.String("url", finalDns))

	//gormDb无需使用.session，它Open出来就是一个链式安全的实例
	db, err := gorm.Open(mysql.Open(finalDns), config)
	if err != nil {
		logger.Fatal("gorm连接数据库失败", zap.Error(err))
		return nil
	}
	err = SetGormThread(db, allConfig)
	if err != nil {
		logger.Fatal("设置gorm协成池失败", zap.Error(err))
		return nil
	}

	logger.Info("连接mysql成功")
	return db
}
