package storage

import (
	"github.com/jackc/pgx"
	"github.com/sirupsen/logrus"
)

type dbLogger struct {
	l logrus.FieldLogger
}

func DBLogger(l logrus.FieldLogger) dbLogger {
	return dbLogger{
		l: l,
	}
}

func (d dbLogger) Log(level pgx.LogLevel, msg string, data map[string]interface{}) {
	var log func(string, ...interface{})
	switch level {
	case pgx.LogLevelNone:
		fallthrough
	case pgx.LogLevelTrace:
		fallthrough
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
