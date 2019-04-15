package logger

import (
	"os"

	"github.com/zdnscloud/cement/log"
)

var gLogger log.Logger

func InitLogger() {
	gLogger = log.NewLog4jConsoleLogger(log.Debug)
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

func Fatal(fmt string, args ...interface{}) {
	GetLogger().Error(fmt, args...)
	os.Exit(1)
}
