package log4go

import (
	"fmt"
	"io"
	"os"
)

var stdout io.Writer = os.Stdout

type ConsoleLogWriter struct {
	format string
	rec    chan *LogRecord
}

func NewConsoleLogWriter(fmt string) *ConsoleLogWriter {
	w := &ConsoleLogWriter{
		format: fmt,
		rec:    make(chan *LogRecord, LogBufferLength),
	}
	go w.run(stdout)
	return w
}

func (w *ConsoleLogWriter) run(out io.Writer) {
	for {
		rec, ok := <-w.rec
		if !ok {
			return
		}

		fmt.Fprint(out, FormatLogRecord(w.format, rec))
	}
}

func (w *ConsoleLogWriter) LogWrite(rec *LogRecord) {
	w.rec <- rec
}

func (w *ConsoleLogWriter) Close() {
	close(w.rec)
}
