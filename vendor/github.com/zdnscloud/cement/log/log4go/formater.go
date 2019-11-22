package log4go

import (
	"bytes"
	"fmt"
	"time"
)

const (
	FORMAT_DEFAULT = "[%D %T] [%L] (%S) %M"
	FORMAT_SHORT   = "[%t %d] [%L] %M"
	FORMAT_ABBREV  = "[%L] %M"
)

type LogFormater interface {
	Format(rec *LogRecord) string
}

type DefaultFormater struct {
	format string
}

func NewDefaultFormater(fmt string) LogFormater {
	return &DefaultFormater{
		format: fmt,
	}
}

var formatCache = &formatCacheType{}

type formatCacheType struct {
	LastUpdateSeconds    int64
	shortTime, shortDate string
	longTime, longDate   string
}

// Known format codes:
// %T - Time (15:04:05 MST)
// %t - Time (15:04)
// %D - Date (2006/01/02)
// %d - Date (01/02/06)
// %L - Level (FNST, FINE, DEBG, TRAC, WARN, EROR, CRIT)
// %S - Source
// %M - Message
// Ignores unknown formats
// Recommended: "[%D %T] [%L] (%S) %M"
func (f *DefaultFormater) Format(rec *LogRecord) string {
	if rec == nil {
		return "<nil>"
	}
	if len(f.format) == 0 {
		return ""
	}

	out := bytes.NewBuffer(make([]byte, 0, 64))
	secs := rec.Created.UnixNano() / 1e9

	cache := *formatCache
	if cache.LastUpdateSeconds != secs {
		month, day, year := rec.Created.Month(), rec.Created.Day(), rec.Created.Year()
		hour, minute, second := rec.Created.Hour(), rec.Created.Minute(), rec.Created.Second()
		zone, _ := rec.Created.Zone()
		updated := &formatCacheType{
			LastUpdateSeconds: secs,
			shortTime:         fmt.Sprintf("%02d:%02d", hour, minute),
			shortDate:         fmt.Sprintf("%02d/%02d/%02d", month, day, year%100),
			longTime:          fmt.Sprintf("%02d:%02d:%02d %s", hour, minute, second, zone),
			longDate:          fmt.Sprintf("%04d/%02d/%02d", year, month, day),
		}
		cache = *updated
		formatCache = updated
	}

	// Split the string into pieces by % signs
	pieces := bytes.Split([]byte(f.format), []byte{'%'})

	// Iterate over the pieces, replacing known formats
	for i, piece := range pieces {
		if i > 0 && len(piece) > 0 {
			switch piece[0] {
			case 'T':
				out.WriteString(cache.longTime)
			case 't':
				out.WriteString(cache.shortTime)
			case 'D':
				out.WriteString(cache.longDate)
			case 'd':
				out.WriteString(cache.shortDate)
			case 'L':
				out.WriteString(levelStrings[rec.Level])
			case 'S':
				out.WriteString(rec.Source)
			case 'M':
				out.WriteString(rec.Message)
			}
			if len(piece) > 1 {
				out.Write(piece[1:])
			}
		} else if len(piece) > 0 {
			out.Write(piece)
		}
	}
	out.WriteByte('\n')

	return out.String()
}

var _ LogFormater = &DefaultFormater{}

type ISO3339Formator struct{}

func NewISO3339Formator() *ISO3339Formator {
	return &ISO3339Formator{}
}

func (f *ISO3339Formator) Format(rec *LogRecord) string {
	if rec == nil {
		return "<nil>"
	}
	out := bytes.NewBuffer(make([]byte, 0, 64))
	t := rec.Created.UTC().Format(time.RFC3339)

	out.WriteString(t)
	out.WriteString(" [")
	out.WriteString(levelStrings[rec.Level])
	out.WriteString("] ")
	out.WriteString(rec.Message)
	out.WriteByte('\n')
	return out.String()
}

var _ LogFormater = &ISO3339Formator{}
