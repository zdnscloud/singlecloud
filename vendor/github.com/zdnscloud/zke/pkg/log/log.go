package log

import (
	"context"
	"fmt"
	"os"

	cementlog "github.com/zdnscloud/cement/log"
)

const (
	zkeLogKey = "zke-logger"
)

var LogLevel cementlog.LogLevel = cementlog.Info

func SetLogger(ctx context.Context, logger cementlog.Logger) (context.Context, error) {
	if logger == nil {
		return ctx, fmt.Errorf("logger can't be nil")
	}
	return context.WithValue(ctx, zkeLogKey, logger), nil
}

func getLogger(ctx context.Context) cementlog.Logger {
	logger := ctx.Value(zkeLogKey).(cementlog.Logger)
	return logger
}

func Debugf(ctx context.Context, msg string, args ...interface{}) {
	getLogger(ctx).Debug(msg, args...)
}

func Infof(ctx context.Context, msg string, args ...interface{}) {
	getLogger(ctx).Info(msg, args...)
}

func Warnf(ctx context.Context, msg string, args ...interface{}) {
	getLogger(ctx).Warn(msg, args...)
}

func Errorf(ctx context.Context, msg string, args ...interface{}) {
	getLogger(ctx).Error(msg, args...)
}

func Fatal(ctx context.Context, args ...interface{}) {
	getLogger(ctx).Error("", args...)
	getLogger(ctx).Close()
	os.Exit(1)
}
