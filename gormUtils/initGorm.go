package gormUtils

import (
	log "log"
	"log/slog"
	"time"

	"github.com/nogolang/common-utils-go/configUtils"
	slogGorm "github.com/orandin/slog-gorm"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// 初始化gorm的配置
func getGormConfigCommon(logger *slog.Logger, allConfig *configUtils.CommonConfig) *gorm.Config {
	//var gormLogLevel gormlogger.LogLevel
	//switch allConfig.Gorm.LogLevel {
	//case "info":
	//	gormLogLevel = gormlogger.Info
	//case "warn":
	//	gormLogLevel = gormlogger.Warn
	//case "error":
	//	gormLogLevel = gormlogger.Error
	//default:
	//	gormLogLevel = gormlogger.Info
	//}
	var config = &gorm.Config{
		//适配zap
		//Logger: gormZap.NewGormZap(logger, gormLogLevel, time.Duration(allConfig.Gorm.SlowSqlMillSecond)*time.Millisecond), //gorm适配zap

		//适配slog
		Logger: slogGorm.New(slogGorm.WithHandler(logger.Handler()),
			slogGorm.WithSlowThreshold(time.Duration(allConfig.Gorm.SlowSqlMillSecond)*time.Second)),
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

func SetGormThread(db *gorm.DB, allConfig *configUtils.CommonConfig) error {
	raw, err := db.DB()
	if err != nil {
		return err
	}

	//设置最大连接数，需要同时设置数据库本身
	raw.SetMaxOpenConns(allConfig.Gorm.MaxOpenConn)
	return nil
}

// NewGormConfig logger由外部注入进来
func NewGorm(logger *slog.Logger, allConfig *configUtils.CommonConfig) *gorm.DB {
	config := getGormConfigCommon(logger, allConfig)
	var db *gorm.DB
	finalDns := allConfig.Gorm.Url
	if allConfig.Gorm.DatabaseType == "" || allConfig.Gorm.DatabaseType == "mysql" {
		//gormDb无需使用.session，它Open出来就是一个链式安全的实例
		var err error
		db, err = gorm.Open(mysql.Open(finalDns), config)
		if err != nil {
			log.Fatal("gorm连接数据库失败", zap.Error(err))
			return nil
		}
	} else if allConfig.Gorm.DatabaseType == "postgres" {
		var err error
		db, err = gorm.Open(postgres.Open(finalDns), config)
		if err != nil {
			log.Fatal("gorm连接数据库失败", zap.Error(err))
			return nil
		}
	} else {
		log.Fatal("不支持的数据库类型", zap.String("databaseType", allConfig.Gorm.DatabaseType))
		return nil
	}

	err := SetGormThread(db, allConfig)
	if err != nil {
		log.Fatal("设置gorm协成池失败", zap.Error(err))
		return nil
	}

	log.Println("连接数据库成功")
	return db
}
