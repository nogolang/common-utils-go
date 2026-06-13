package dtmUtils

import (
	"log/slog"

	"github.com/nogolang/common-utils-go/configUtils"
)
import (
	dtmClientLogger "github.com/dtm-labs/dtm/client/dtmcli/logger"
)

type DtmLogger struct {
	logger *slog.Logger
}

func (receiver *DtmLogger) Debugf(format string, args ...interface{}) {
	receiver.logger.Debug(format, slog.Any("args", args))
}
func (receiver *DtmLogger) Infof(format string, args ...interface{}) {

	receiver.logger.Info(format, slog.Any("args", args))
}
func (receiver *DtmLogger) Warnf(format string, args ...interface{}) {
	receiver.logger.Warn(format, slog.Any("args", args))
}
func (receiver *DtmLogger) Errorf(format string, args ...interface{}) {
	receiver.logger.Error(format, slog.Any("args", args))
}
func NewDtmLogger(common *configUtils.CommonConfig, logger *slog.Logger) *DtmLogger {
	var dtmlog DtmLogger
	dtmlog.logger = logger //赋予slog
	dtmClientLogger.WithLogger(&dtmlog)
	if common.Dtm.LogLevel == "" {
		common.Dtm.LogLevel = "info"
	}
	dtmClientLogger.InitLog(getDtmLevel(common.Dtm.LogLevel))
	return &dtmlog
}
func getDtmLevel(str string) string {
	switch str {
	case "debug":
		return "debug"
	case "info":
		return "info"
	case "warn":
		return "warn"
	case "error":
		return "error"
	}
	return "info"
}
