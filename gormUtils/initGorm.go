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
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// 初始化gorm的配置
//
// customLogger 可选：传入自定义 gormlogger.Interface（如过滤 SELECT 的 logger）；
// 不传时用默认 slogGorm（TraceAll + Error + Slow）。
// 这给「shop-main-task 等后台 worker」一种能力：过滤周期性扫库 SELECT 日志。
func getGormConfigCommon(logger *slog.Logger, allConfig *configUtils.CommonConfig, customLogger gormlogger.Interface) *gorm.Config {
	var gormLoggerInst gormlogger.Interface
	if customLogger != nil {
		gormLoggerInst = customLogger
	} else {
		gormLoggerInst = slogGorm.New(slogGorm.WithHandler(logger.Handler()),
			slogGorm.WithTraceAll(), // trace all messages，此时才会打印默认的
			slogGorm.WithSlowThreshold(time.Duration(allConfig.Gorm.SlowSqlMillSecond)*time.Millisecond),
			slogGorm.WithSourceField(""), //不打印文件行，因为打印的是插件的行数，不是业务的
			slogGorm.SetLogLevel(slogGorm.DefaultLogType, slog.LevelInfo),
			slogGorm.SetLogLevel(slogGorm.ErrorLogType, slog.LevelError),
			slogGorm.SetLogLevel(slogGorm.SlowQueryLogType, slog.LevelWarn),
			slogGorm.WithContextValue("traceId", "traceId"),
		)
	}

	var config = &gorm.Config{
		//适配slog
		Logger: gormLoggerInst,
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

// NewGorm logger由外部注入进来，使用默认 slog-gorm logger
//
// 业务模块（admin/api/business）用这个版本：所有 SQL 打 INFO 日志。
// 后台 worker（shop-main-task）用 NewGormWithLogger 传自定义 logger 过滤 SELECT。
func NewGorm(logger *slog.Logger, allConfig *configUtils.CommonConfig) *gorm.DB {
	return NewGormWithLogger(logger, allConfig, nil)
}

// NewGormWithLogger 带自定义 gorm logger 的版本
//
// customLogger 为 nil 时用默认 slog-gorm logger（同 NewGorm 行为）。
// 设计动机（见 shop-当前需求.md「task 模块日志噪音」）：
//   - task 模块传 NewSelectFilteringLogger 过滤周期性扫库 SELECT 日志
//   - 业务模块不传，保留原行为
func NewGormWithLogger(logger *slog.Logger, allConfig *configUtils.CommonConfig, customLogger gormlogger.Interface) *gorm.DB {
	config := getGormConfigCommon(logger, allConfig, customLogger)
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
