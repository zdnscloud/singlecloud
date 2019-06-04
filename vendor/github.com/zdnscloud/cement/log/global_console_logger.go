package log

import (
	"os"
)

var gLogger Logger

func InitLogger(level LogLevel) {
	gLogger = NewLog4jConsoleLogger(level)
}

func Debugf(fmt string, args ...interface{}) {
	gLogger.Debug(fmt, args...)
}

func Infof(fmt string, args ...interface{}) {
	gLogger.Info(fmt, args...)
}

func Warnf(fmt string, args ...interface{}) error {
	return gLogger.Warn(fmt, args...)
}

func Errorf(fmt string, args ...interface{}) error {
	return gLogger.Error(fmt, args...)
}

func Fatalf(fmt string, args ...interface{}) {
	gLogger.Error(fmt, args...)
	gLogger.Close()
	os.Exit(1)
}

func CloseLogger() {
	gLogger.Close()
}
