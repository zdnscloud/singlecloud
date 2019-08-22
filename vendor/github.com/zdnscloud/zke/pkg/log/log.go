package log

import (
	"context"
	"os"

	cementlog "github.com/zdnscloud/cement/log"
)

var ZKELogger cementlog.Logger
var DefaultLogLevel = cementlog.Info

func InitConsoleLog() {
	ZKELogger = cementlog.NewLog4jConsoleLogger(DefaultLogLevel)
}

func InitChannelLog(l cementlog.Logger) {
	ZKELogger = l
}

func Debugf(msg string, args ...interface{}) {
	ZKELogger.Debug(msg, args...)
}

func Infof(ctx context.Context, msg string, args ...interface{}) {
	ZKELogger.Info(msg, args...)
}

func Warnf(ctx context.Context, msg string, args ...interface{}) {
	ZKELogger.Warn(msg, args...)
}

func Errorf(ctx context.Context, msg string, args ...interface{}) {
	ZKELogger.Error(msg, args...)
}

func Fatal(args ...interface{}) {
	ZKELogger.Error("", args...)
	ZKELogger.Close()
	os.Exit(1)
}
