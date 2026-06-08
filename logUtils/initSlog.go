package logUtils

import (
	"log"
	"log/slog"

	"github.com/nogolang/common-utils-go/configUtils"
	slogzap "github.com/samber/slog-zap/v2"
)

var slogLevel *slog.LevelVar

func InitSlogLevel() {
	slogLevel = NewSlogLevel(configUtils.GetCommonConfig())
}
func GetSlogLevel() *slog.LevelVar {
	return slogLevel
}
func NewSlogLevel(commonConfig *configUtils.CommonConfig) *slog.LevelVar {
	return &slog.LevelVar{}
}

var logger *slog.Logger

func InitSlog() {
	logger = NewSlog(configUtils.GetCommonConfig(), GetSlogLevel())
}
func GetSlog() *slog.Logger {
	return logger
}
func NewSlog(commonConfig *configUtils.CommonConfig, level *slog.LevelVar) *slog.Logger {
	var nowUse string
	//默认使用zap
	if commonConfig.Log.Use == "" {
		nowUse = "zap"
	}

	var logger *slog.Logger
	switch nowUse {
	case "zap":
		zap := NewZapConfig(commonConfig, slogzap.LogLevels[level.Level()])
		logger = slog.New(slogzap.Option{Level: level, Logger: zap}.NewZapHandler())
	default:
		log.Fatal("未定义的日志使用方式")
		return nil
	}

	slog.SetDefault(logger)
	return logger
}
