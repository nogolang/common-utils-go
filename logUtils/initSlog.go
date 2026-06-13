package logUtils

import (
	"context"
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

func NewSlog(commonConfig *configUtils.CommonConfig, level *slog.LevelVar) *slog.Logger {
	var nowUse string
	//默认使用zap
	if commonConfig.Log.Use == "" {
		nowUse = "zap"
	}

	//traceId自动打印
	attrFromContext := []func(ctx context.Context) []slog.Attr{
		func(ctx context.Context) []slog.Attr {
			return []slog.Attr{
				slog.Any("traceId", ctx.Value("traceId")),
			}
		},
	}

	var logger *slog.Logger
	switch nowUse {
	case "zap":
		zap := NewZapConfig(commonConfig, slogzap.LogLevels[level.Level()])
		logger = slog.New(slogzap.Option{Level: level, Logger: zap, AttrFromContext: attrFromContext}.
			NewZapHandler())
	default:
		log.Fatal("未定义的日志使用方式")
		return nil
	}
	slog.SetDefault(logger)
	return logger
}
