package logUtils

import (
	"testing"

	"go.uber.org/zap"
)

func Test_zap(t *testing.T) {
	cfg := &LogConfig{
		Level:       "info",
		HiddenField: []string{"password"},
	}
	level := NewZapAtomicLevel(cfg)
	logger := NewZapConfig(cfg, level.Level())
	logger.Info("hello world", zap.String("password", "123456"))
}
