package logUtils

import (
	"context"
	"log"
	"log/slog"

	slogzap "github.com/samber/slog-zap/v2"
)

var slogLevel *slog.LevelVar

// InitSlogLevel 手动设置全局日志级别。
func InitSlogLevel(level *slog.LevelVar) {
	slogLevel = level
}

func GetSlogLevel() *slog.LevelVar {
	return slogLevel
}

// NewSlogLevel 按配置解析日志级别（空/非法一律回落 info）
func NewSlogLevel(cfg *LogConfig) *slog.LevelVar {
	level := new(slog.LevelVar)
	if cfg == nil {
		level.Set(slog.LevelInfo)
		return level
	}
	switch cfg.Level {
	case "debug":
		level.Set(slog.LevelDebug)
	case "warn":
		level.Set(slog.LevelWarn)
	case "error":
		level.Set(slog.LevelError)
	default:
		//info 及一切未识别值（含空）都按 info
		level.Set(slog.LevelInfo)
	}
	return level
}

func NewSlog(cfg *LogConfig, level *slog.LevelVar) *slog.Logger {
	var nowUse string
	//默认使用zap
	if cfg != nil && cfg.Use != "" {
		nowUse = cfg.Use
	} else {
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
		zap := NewZapConfig(cfg, slogzap.LogLevels[level.Level()])
		logger = slog.New(slogzap.Option{Level: level, Logger: zap, AddSource: true, AttrFromContext: attrFromContext}.
			NewZapHandler())
	default:
		log.Fatal("未定义的日志使用方式")
		return nil
	}
	slog.SetDefault(logger)
	return logger
}
