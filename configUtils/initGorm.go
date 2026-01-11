package configUtils

import (
	"time"

	//这是我们自己的包，因为要导入私人包的原因，把GOPRIVATE=设置为了自己的包
	//但是此时就不走代理了，所以我们要在环境变量里设置一下，让它置空
	"github.com/nogolang/gorm-zap/gormZap"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// 初始化gorm的配置
func getGormConfigCommon(logger *zap.Logger, allConfig *AllConfig) *gorm.Config {
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
		//不自动创建外键
		DisableForeignKeyConstraintWhenMigrating: allConfig.Gorm.AutoCreateForeignKey,
	}
	return config
}

func SetGormThread(db *gorm.DB, allConfig *AllConfig) error {
	rawDb, err := db.DB()
	if err != nil {
		return err
	}

	//设置最大连接数，需要同时设置数据库本身
	rawDb.SetMaxOpenConns(allConfig.Gorm.MaxOpenConn)
	return nil
}

// NewGormConfig logger由外部注入进来
func NewGorm(logger *zap.Logger, allConfig *AllConfig) *gorm.DB {
	config := getGormConfigCommon(logger, allConfig)
	//gormDb无需使用.session，它Open出来就是一个链式安全的实例
	db, err := gorm.Open(mysql.Open(allConfig.Gorm.Url), config)
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
