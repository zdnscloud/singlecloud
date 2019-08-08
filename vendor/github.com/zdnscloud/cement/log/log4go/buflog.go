package log4go

type BufLogWriter struct {
	format       string
	rec          chan *LogRecord
	logCh        chan string
	bufCloseSync chan struct{}
}

func NewBufLogWriter(logChLength uint, fmt string) *BufLogWriter {
	w := &BufLogWriter{
		format:       fmt,
		rec:          make(chan *LogRecord, LogBufferLength),
		logCh:        make(chan string, logChLength),
		bufCloseSync: make(chan struct{}),
	}
	go w.run()
	return w
}

func (w *BufLogWriter) run() {
	for {
		rec, ok := <-w.rec
		if !ok {
			close(w.logCh)
			close(w.bufCloseSync)
			return
		}
		log := FormatLogRecord(w.format, rec)

		select {
		case w.logCh <- log:
		default:
			<-w.logCh
			w.logCh <- log
		}
	}
}

func (w *BufLogWriter) LogWrite(rec *LogRecord) {
	w.rec <- rec
}

func (w *BufLogWriter) Close() {
	close(w.rec)
	<-w.bufCloseSync
}

func NewBufLogger(logChLength uint, lvl level, fmt string) (Logger, chan string) {
	bufLogWriter := NewBufLogWriter(logChLength, fmt)

	return Logger{
		"buf": &Filter{lvl, bufLogWriter},
	}, bufLogWriter.logCh
}
