package log

import (
	"git.sr.ht/~mariusor/lw"
	"github.com/jackc/pgx"
)

type pgxLogger struct {
	l lw.Logger
}

func NewPgxLogger(l lw.Logger) pgxLogger {
	return pgxLogger{
		l: l,
	}
}

func (d pgxLogger) Log(level pgx.LogLevel, msg string, data map[string]interface{}) {
	var log func(string, ...interface{})
	switch level {
	case pgx.LogLevelNone:
		fallthrough
	case pgx.LogLevelTrace:
		log = d.l.WithContext(data).Debugf
	case pgx.LogLevelDebug:
		log = d.l.WithContext(data).Debugf
	case pgx.LogLevelInfo:
		log = d.l.WithContext(data).Infof
	case pgx.LogLevelWarn:
		log = d.l.WithContext(data).Warnf
	case pgx.LogLevelError:
		log = d.l.WithContext(data).Errorf
	}
	log(msg)
}
