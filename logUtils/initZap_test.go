package logUtils

import (
	"testing"

	"github.com/nogolang/common-utils-go/configUtils"
	"go.uber.org/zap"
)

func Test_zap(t *testing.T) {
	common := configUtils.CommonConfig{
		Log: &configUtils.LogConfig{
			Level:       "info",
			HiddenField: []string{"password"},
		},
	}
	level := NewZapAtomicLevel(&common)
	logger := NewZapConfig(&common, level.Level())
	logger.Info("hello world", zap.String("password", "123456"))
}
