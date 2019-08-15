package log

import (
	"github.com/jackc/pgx"
	"github.com/sirupsen/logrus"
)

type pgxLogger struct {
	l logrus.FieldLogger
}

func NewPgxLogger(l logrus.FieldLogger) pgxLogger {
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
		log = d.l.WithFields(data).Tracef
	case pgx.LogLevelDebug:
		log = d.l.WithFields(data).Debugf
	case pgx.LogLevelInfo:
		log = d.l.WithFields(data).Infof
	case pgx.LogLevelWarn:
		log = d.l.WithFields(data).Warnf
	case pgx.LogLevelError:
		log = d.l.WithFields(data).Errorf
	}
	log(msg)
}
