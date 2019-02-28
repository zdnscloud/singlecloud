package log4go

import (
	"errors"
	"fmt"
	"runtime"
	"time"
)

type level int

const (
	DEBUG level = iota
	INFO
	WARNING
	ERROR
)

var (
	levelStrings = [...]string{"DEBG", "INFO", "WARN", "EROR"}
)

func (l level) String() string {
	if l < 0 || int(l) > len(levelStrings) {
		return "UNKNOWN"
	}
	return levelStrings[int(l)]
}

var (
	// LogBufferLength specifies how many log messages a particular log4go
	// logger can buffer at a time before writing them.
	LogBufferLength = 32
)

type LogRecord struct {
	Level   level     // The log level
	Created time.Time // The time at which the log message was created (nanoseconds)
	Source  string    // The message source
	Message string    // The log message
}

type LogWriter interface {
	LogWrite(rec *LogRecord)
	Close()
}

type Filter struct {
	Level level
	LogWriter
}

type Logger map[string]*Filter

func NewDefaultLogger(lvl level, fmt string) Logger {
	return Logger{
		"stdout": &Filter{lvl, NewConsoleLogWriter("")},
	}
}

func (log Logger) Close() {
	for name, filt := range log {
		filt.Close()
		delete(log, name)
	}
}

func (log Logger) AddFilter(name string, lvl level, writer LogWriter) Logger {
	log[name] = &Filter{lvl, writer}
	return log
}

func (log Logger) intLogf(lvl level, format string, args ...interface{}) {
	skip := true

	// Determine if any logging will be done
	for _, filt := range log {
		if lvl >= filt.Level {
			skip = false
			break
		}
	}
	if skip {
		return
	}

	// Determine caller func
	pc, _, lineno, ok := runtime.Caller(3)
	src := ""
	if ok {
		src = fmt.Sprintf("%s:%d", runtime.FuncForPC(pc).Name(), lineno)
	}

	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}

	// Make the log record
	rec := &LogRecord{
		Level:   lvl,
		Created: time.Now(),
		Source:  src,
		Message: msg,
	}

	// Dispatch the logs
	for _, filt := range log {
		if lvl < filt.Level {
			continue
		}
		filt.LogWrite(rec)
	}
}

// Send a log message with manual level, source, and message.
func (log Logger) Log(lvl level, source, message string) {
	skip := true

	// Determine if any logging will be done
	for _, filt := range log {
		if lvl >= filt.Level {
			skip = false
			break
		}
	}
	if skip {
		return
	}

	// Make the log record
	rec := &LogRecord{
		Level:   lvl,
		Created: time.Now(),
		Source:  source,
		Message: message,
	}

	// Dispatch the logs
	for _, filt := range log {
		if lvl < filt.Level {
			continue
		}
		filt.LogWrite(rec)
	}
}

func (log Logger) Logf(lvl level, format string, args ...interface{}) {
	log.intLogf(lvl, format, args...)
}

func (log Logger) Debug(format string, args ...interface{}) {
	log.intLogf(DEBUG, format, args...)
}

func (log Logger) Info(format string, args ...interface{}) {
	log.intLogf(INFO, format, args...)
}

func (log Logger) Warn(format string, args ...interface{}) error {
	msg := fmt.Sprintf(format, args...)
	log.intLogf(WARNING, msg)
	return errors.New(msg)
}

func (log Logger) Error(format string, args ...interface{}) error {
	msg := fmt.Sprintf(format, args...)
	log.intLogf(ERROR, msg)
	return errors.New(msg)
}
