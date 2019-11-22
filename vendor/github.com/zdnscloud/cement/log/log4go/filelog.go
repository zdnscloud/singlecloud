package log4go

import (
	"fmt"
	"os"
)

const (
	DefaultFileCount   = 10
	DefaultMaxFileSize = 100 * 1000 * 1000 //100m byte
)

type FileLogWriter struct {
	rec chan *LogRecord
	rot chan bool

	// The opened file
	filename string
	file     *os.File

	// The logging format
	formater LogFormater

	// Rotate at size
	maxsize int
	cursize int

	//when file exceed the count
	//old file will be deleted
	maxfilecount  int
	fileCloseSync chan struct{}
}

func (w *FileLogWriter) LogWrite(rec *LogRecord) {
	w.rec <- rec
}

func (w *FileLogWriter) Close() {
	close(w.rec)
	if w.fileCloseSync != nil {
		<-w.fileCloseSync
	}
}

// NewFileLogWriter creates a new LogWriter which writes to the given file and
// has rotation enabled if rotate is true.
//
// If rotate is true, any time a new log file is opened, the old one is renamed
// with a .### extension to preserve it.  The various Set* methods can be used
// to configure log rotation based on lines, size, and daily.
//
// The standard log-line format is:
//   [%D %T] [%L] (%S) %M
func NewFileLogWriter(fname string, formater LogFormater, maxsize, maxfilecount int) (*FileLogWriter, error) {
	if maxfilecount < 1 {
		maxfilecount = DefaultFileCount
	}

	if maxsize <= 0 {
		maxsize = DefaultMaxFileSize
	}

	w := &FileLogWriter{
		rec:          make(chan *LogRecord, LogBufferLength),
		filename:     fname,
		formater:     formater,
		maxsize:      maxsize,
		maxfilecount: maxfilecount,
	}

	// open the file for the first time
	if err := w.init(); err != nil {
		fmt.Fprintf(os.Stderr, "init log failed(%q): %s\n", w.filename, err)
		return nil, err
	}

	go func() {
		var err error
		w.fileCloseSync = make(chan struct{}, 1)
		defer func() {
			if w.file != nil {
				w.file.Close()
			}

			if err == nil {
				w.fileCloseSync <- struct{}{}
			} else {
				close(w.rec)
			}
		}()

		for {
			var recordLen int
			select {
			case rec, ok := <-w.rec:
				if !ok {
					return
				}
				if w.maxsize > 0 && w.cursize >= w.maxsize {
					if err = w.rotate(); err != nil {
						fmt.Fprintf(os.Stderr, "rotate log failed(%q): %s\n", w.filename, err)
						return
					}
				}

				recordLen, err = fmt.Fprint(w.file, w.formater.Format(rec))
				if err != nil {
					fmt.Fprintf(os.Stderr, "format log record failed(%q): %s\n", w.filename, err)
					return
				}

				w.cursize += recordLen
			}
		}
	}()

	return w, nil
}

func (w *FileLogWriter) init() error {
	fd, err := os.OpenFile(w.filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)
	if err != nil {
		return err
	}
	w.file = fd

	if info, err := os.Lstat(w.filename); err != nil {
		return err
	} else {
		w.cursize = int(info.Size())
		return nil
	}
}

// If this is called in a threaded context, it MUST be synchronized
func (w *FileLogWriter) rotate() error {
	w.file.Close()

	if w.maxfilecount > 2 {
		for num := w.maxfilecount - 2; num > 0; num-- {
			fname := w.filename + fmt.Sprintf(".%03d", num)
			if _, err := os.Lstat(fname); err == nil {
				older_fname := w.filename + fmt.Sprintf(".%03d", num+1)
				err = os.Rename(fname, older_fname)
				if err != nil {
					return fmt.Errorf("Rotate: %s\n", err)
				}
			}
		}
	}

	if w.maxfilecount > 1 {
		_, err := os.Lstat(w.filename)
		if err == nil { // file exists
			err = os.Rename(w.filename, w.filename+".001")
			if err != nil {
				return fmt.Errorf("Rotate: %s\n", err)
			}
		}
	}

	// Open the log file
	fd, err := os.OpenFile(w.filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)
	if err != nil {
		return err
	}
	w.file = fd
	w.cursize = 0

	return nil
}
