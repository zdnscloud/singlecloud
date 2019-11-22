package log4go

import (
	"fmt"
	"io"
	"os"
)

var stdout io.Writer = os.Stdout

type ConsoleLogWriter struct {
	formater      LogFormater
	rec           chan *LogRecord
	termCloseSync chan struct{}
}

func NewConsoleLogWriter(formater LogFormater) *ConsoleLogWriter {
	w := &ConsoleLogWriter{
		formater:      formater,
		rec:           make(chan *LogRecord, LogBufferLength),
		termCloseSync: make(chan struct{}),
	}
	go w.run(stdout)
	return w
}

func (w *ConsoleLogWriter) run(out io.Writer) {
	for {
		rec, ok := <-w.rec
		if !ok {
			close(w.termCloseSync)
			return
		}

		fmt.Fprint(out, w.formater.Format(rec))
	}
}

func (w *ConsoleLogWriter) LogWrite(rec *LogRecord) {
	w.rec <- rec
}

func (w *ConsoleLogWriter) Close() {
	close(w.rec)
	<-w.termCloseSync
}
