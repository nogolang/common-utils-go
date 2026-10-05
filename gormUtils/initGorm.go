package gormUtils

import (
	log "log"
	"log/slog"
	"strings"
	"time"

	slogGorm "github.com/orandin/slog-gorm"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// ParseSqlLogLevel gorm.logLevel 配置值 → SQL 常规日志的 slog 标签。
//
// 语义（**对齐常规日志直觉**，2026-10-01 用户纠偏定稿）：值 = 想看到的 SQL 日志下限——
//   - info / debug → 每条 SQL 都显示（贴 Info 标签）
//   - warn / error / 缺省 / 非法 → 常规 SQL 不显示（贴 Debug 标签；维持治刷屏默认）
//   - debug 在 SQL 维度没有额外明细可展开，与 info 等价
//
// ⚠️ 机制说明（为什么"更少日志"反而贴更低的标签）：slog-gorm 的 SetLogLevel 是给日志条目
// **贴级别标签**，不是设显示门槛——条目是否输出由**进程 slog 门槛**（yaml log.level，恒 ≥ info）
// 决定：标签 ≥ 门槛 → 显示；标签 < 门槛 → 隐藏。所以「隐藏常规 SQL」只能贴 Debug 实现；
// 千万别把 warn/error 映射成贴 Warn/Error 标签——那反而一定显示，还会和真正的慢查询/错误混在一起。
func ParseSqlLogLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "info", "debug":
		return slog.LevelInfo
	default: // warn / error / 空 / 非法
		return slog.LevelDebug
	}
}

// getGormConfigCommon 组装 gorm.Config
//
// customLogger 可选：传入自定义 gormlogger.Interface（如过滤 SELECT 的 logger）；
// 不传时用默认 slogGorm（TraceAll + Error + Slow）。
// 这给「后台 worker」一种能力：过滤周期性扫库 SELECT 日志。
func getGormConfigCommon(logger *slog.Logger, cfg *GormConfig, customLogger gormlogger.Interface) *gorm.Config {
	var gormLoggerInst gormlogger.Interface
	if customLogger != nil {
		gormLoggerInst = customLogger
	} else {
		gormLoggerInst = slogGorm.New(slogGorm.WithHandler(logger.Handler()),
			slogGorm.WithTraceAll(), // trace all messages，此时才会打印默认的
			slogGorm.WithSlowThreshold(time.Duration(cfg.SlowSqlMillSecond)*time.Millisecond),
			slogGorm.WithSourceField(""), //不打印文件行，因为打印的是插件的行数，不是业务的
			// SQL 常规日志级别读 gorm.logLevel（2026-10-01 接通；语义=想看到的 SQL 日志下限，
			// 对齐常规日志直觉）：info/debug=每条 SQL 都显示；warn/error/缺省=常规 SQL 不显示
			//（慢查询 Warn、SQL 错误 Error 恒浮出，不受此字段影响）。见 ParseSqlLogLevel 注释。
			slogGorm.SetLogLevel(slogGorm.DefaultLogType, ParseSqlLogLevel(cfg.LogLevel)),
			slogGorm.SetLogLevel(slogGorm.ErrorLogType, slog.LevelError),
			slogGorm.SetLogLevel(slogGorm.SlowQueryLogType, slog.LevelWarn),
			slogGorm.WithContextValue("traceId", "traceId"),
		)
	}

	var config = &gorm.Config{
		//适配slog
		Logger: gormLoggerInst,
		NamingStrategy: schema.NamingStrategy{
			SingularTable: cfg.SingularTable,
		},
		//是否把驱动错误翻译成 gorm 的标准错误（ErrDuplicatedKey / ErrRecordNotFound…）
		TranslateError: cfg.TransError,
		//是否关闭自动创建外键
		DisableForeignKeyConstraintWhenMigrating: cfg.DisableAutoCreateForeignKey,
	}
	return config
}

func SetGormThread(db *gorm.DB, cfg *GormConfig) error {
	raw, err := db.DB()
	if err != nil {
		return err
	}

	//设置最大连接数，需要同时设置数据库本身
	raw.SetMaxOpenConns(cfg.MaxOpenConn)
	// 空闲连接数（2026-09-20 补）：此前**从未设置** → database/sql 默认仅 2，
	// 并发一上来就"建连接→用完关掉→再建"，TCP 握手 + PG 后端进程创建成了主要开销。
	// 取最大连接数的 1/4（不低于 2）而不是同值：本项目 PG 的 max_connections=100 与
	// maxOpenConn 同值，若空闲数也设 100，单进程就能把 PG 连接池占满（psql 都连不进去）。
	idle := cfg.MaxOpenConn / 4
	if idle < 2 {
		idle = 2
	}
	raw.SetMaxIdleConns(idle)
	// 空闲连接超时回收（Go 1.15+）：长时间不用的连接还给 PG，避免"占着连接不干活"
	raw.SetConnMaxIdleTime(5 * time.Minute)
	//连接最长存活（默认 5 分钟）：让 zhparser 词典更新（仅新连接生效）能在窗口内扩散到全池
	life := cfg.ConnMaxLifetimeMinutes
	if life <= 0 {
		life = 5
	}
	raw.SetConnMaxLifetime(time.Duration(life) * time.Minute)
	// 登记进全局池清单（DrainAllIdlePools 用：词典更新等场景需要全进程换新连接）
	registerPool(raw)
	return nil
}

// NewGorm logger由外部注入进来，使用默认 slog-gorm logger
//
// 业务模块（admin/api/business）用这个版本：所有 SQL 打 INFO 日志。
// 后台 worker 用 NewGormWithLogger 传自定义 logger 过滤 SELECT。
func NewGorm(logger *slog.Logger, cfg *GormConfig) *gorm.DB {
	return NewGormWithLogger(logger, cfg, nil)
}

// NewGormWithLogger 带自定义 gorm logger 的版本
//
// customLogger 为 nil 时用默认 slog-gorm logger（同 NewGorm 行为）。
// 设计动机：task 模块传 NewSelectFilteringLogger 过滤周期性扫库 SELECT 日志，
// 业务模块不传，保留原行为。
func NewGormWithLogger(logger *slog.Logger, cfg *GormConfig, customLogger gormlogger.Interface) *gorm.DB {
	config := getGormConfigCommon(logger, cfg, customLogger)
	var db *gorm.DB
	finalDns := cfg.Url
	if cfg.DatabaseType == "" || cfg.DatabaseType == "mysql" {
		//gormDb无需使用.session，它Open出来就是一个链式安全的实例
		var err error
		db, err = gorm.Open(mysql.Open(finalDns), config)
		if err != nil {
			log.Fatal("gorm连接数据库失败", zap.Error(err))
			return nil
		}
	} else if cfg.DatabaseType == "postgres" {
		var err error
		db, err = gorm.Open(postgres.Open(finalDns), config)
		if err != nil {
			log.Fatal("gorm连接数据库失败", zap.Error(err))
			return nil
		}
	} else {
		log.Fatal("不支持的数据库类型", zap.String("databaseType", cfg.DatabaseType))
		return nil
	}

	err := SetGormThread(db, cfg)
	if err != nil {
		log.Fatal("设置gorm协成池失败", zap.Error(err))
		return nil
	}

	log.Println("连接数据库成功")
	return db
}
