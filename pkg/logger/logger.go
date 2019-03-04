package logger

import (
	"github.com/zdnscloud/cement/log"
)

var gLogger log.Logger

func InitLogger() (err error) {
	gLogger = log.NewLog4jConsoleLogger(log.Debug)
	return nil
}

func GetLogger() log.Logger {
	return gLogger
}

func Debug(fmt string, args ...interface{}) {
	GetLogger().Debug(fmt, args...)
}

func Info(fmt string, args ...interface{}) {
	GetLogger().Info(fmt, args...)
}

func Warn(fmt string, args ...interface{}) error {
	return GetLogger().Warn(fmt, args...)
}

func Error(fmt string, args ...interface{}) error {
	return GetLogger().Error(fmt, args...)
}
