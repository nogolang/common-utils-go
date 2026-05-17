package configUtils

import (
	"testing"

	"go.uber.org/zap"
)

func Test_zap(t *testing.T) {
	common := CommonConfig{
		Log: &LogConfig{
			Level:       "info",
			HiddenField: []string{"password"},
		},
	}
	level := NewZapAtomicLevel(&common)
	logger := NewZapConfig(&common, level)
	logger.Info("hello world", zap.String("password", "123456"))
}
