package log

import (
	"context"
	"os"

	cementlog "github.com/zdnscloud/cement/log"
)

const (
	DefaultLogger      = "console"
	ChannelLogger      = "channel"
	DefaultLogChLength = 50
)

var ZKELogger cementlog.Logger
var ZKELoggerKind string = DefaultLogger
var ZKELogLevel = cementlog.Info
var ZKELogCh chan string

func Init() {
	if ZKELoggerKind == ChannelLogger {
		ZKELogger, ZKELogCh = cementlog.NewLog4jBufLogger(DefaultLogChLength, ZKELogLevel)
	} else {
		ZKELogger = cementlog.NewLog4jConsoleLogger(ZKELogLevel)
	}
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
