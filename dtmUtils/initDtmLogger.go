package dtmUtils

import (
	"log/slog"
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
func NewDtmLogger(cfg *DtmConfig, logger *slog.Logger) *DtmLogger {
	var dtmlog DtmLogger
	dtmlog.logger = logger //赋予slog
	dtmClientLogger.WithLogger(&dtmlog)
	logLevel := "info"
	if cfg != nil && cfg.LogLevel != "" {
		logLevel = cfg.LogLevel
	}
	dtmClientLogger.InitLog(getDtmLevel(logLevel))
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
