package logger

import (
	"strings"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/slice"
	"github.com/zdnscloud/singlecloud/config"
)

var gLogger log.Logger

var supportedLevels = []string{
	"debug",
	"info",
	"error",
}

const defaultLogLevelIndex = 1

func InitLogger(conf *config.SingleCloudConf) (err error) {
	logLevel := strings.ToLower(conf.Logger.Level)
	if slice.SliceIndex(supportedLevels, logLevel) == -1 {
		logLevel = supportedLevels[defaultLogLevelIndex]
	}

	gLogger, err = log.NewLog4jLoggerWithFmt(conf.Logger.LogFile,
		log.LogLevel(conf.Logger.Level),
		conf.Logger.FileSize,
		conf.Logger.Versions, "")
	return
}

func GetLogger() log.Logger {
	return gLogger
}
