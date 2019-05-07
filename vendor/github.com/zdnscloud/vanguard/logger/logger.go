package logger

import (
	"strings"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/slice"
	"github.com/zdnscloud/vanguard/config"
)

var globalLogger log.Logger

var supportedLevels = []string{
	"debug",
	"info",
	"error",
}

const defaultLogLevelIndex = 1

func InitLogger(conf *config.VanguardConf) error {
	logLevel := strings.ToLower(conf.Logger.GeneralLog.Level)
	if slice.SliceIndex(supportedLevels, logLevel) == -1 {
		logLevel = supportedLevels[defaultLogLevelIndex]
	}

	var err error
	if conf.Logger.GeneralLog.Enable == false {
		globalLogger = log.NewBlackHole()
	} else if conf.Logger.GeneralLog.Path == "" {
		globalLogger = log.NewLog4jConsoleLogger(log.LogLevel(logLevel))
	} else {
		globalLogger, err = log.NewLog4jLoggerWithFmt(conf.Logger.GeneralLog.Path,
			log.LogLevel(logLevel),
			conf.Logger.GeneralLog.FileSize,
			conf.Logger.GeneralLog.Versions, "")
	}
	return err
}

func GetLogger() log.Logger {
	return globalLogger
}

//for testing
func UseDefaultLogger(level string) {
	if globalLogger == nil {
		globalLogger = log.NewLog4jConsoleLogger(log.LogLevel(level))
	}
}
