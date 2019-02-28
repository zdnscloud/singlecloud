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
