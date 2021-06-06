package log

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/sirupsen/logrus"
)

type Level int8

const (
	PanicLevel Level = iota
	FatalLevel
	ErrorLevel
	WarnLevel
	InfoLevel
	DebugLevel
	TraceLevel
)

var devFormatter = logrus.TextFormatter{
	ForceColors:            false,
	TimestampFormat:        time.StampMilli,
	FullTimestamp:          true,
	DisableSorting:         true,
	DisableLevelTruncation: false,
	PadLevelText:           true,
	QuoteEmptyFields:       false,
}

func New(lvl Level) logrus.FieldLogger {
	l := logrus.New()
	l.SetFormatter(&devFormatter)
	l.Level = logrus.Level(lvl)
	l.Out = os.Stdout
	return l
}

func NewStructuredLogger(logger logrus.FieldLogger) func(next http.Handler) http.Handler {
	return middleware.RequestLogger(&StructuredLogger{logger})
}

type StructuredLogger struct {
	Logger logrus.FieldLogger
}

func (l *StructuredLogger) NewLogEntry(r *http.Request) middleware.LogEntry {
	entry := &StructuredLoggerEntry{}

	ll, ok := l.Logger.(*logrus.Logger)
	if !ok {
		return entry
	}

	entry.Logger = logrus.NewEntry(ll)
	logFields := logrus.Fields{}

	logFields["ts"] = time.Now().UTC().Format(time.RFC1123)

	if reqID := middleware.GetReqID(r.Context()); reqID != "" {
		logFields["id"] = reqID
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	logFields["scheme"] = scheme
	logFields["proto"] = r.Proto
	logFields["method"] = r.Method

	logFields["addr"] = r.RemoteAddr
	logFields["ua"] = r.UserAgent()

	logFields["uri"] = fmt.Sprintf("%s://%s%s", scheme, r.Host, r.RequestURI)

	entry.Logger = entry.Logger.WithFields(logFields)

	return entry
}

type StructuredLoggerEntry struct {
	Logger logrus.FieldLogger
}

func (l *StructuredLoggerEntry) Write(status, bytes int, header http.Header, elapsed time.Duration, extra interface{}) {
	l.Logger = l.Logger.WithFields(logrus.Fields{
		"status":  status,
		"len":     bytes,
		"elapsed": elapsed,
	})

	var logFn func(args ...interface{})
	if status >= 200 && status < 400 {
		logFn = l.Logger.Info
	} else {
		logFn = l.Logger.Error
	}
	logFn(http.StatusText(status))
}

func (l *StructuredLoggerEntry) Panic(v interface{}, stack []byte) {
	l.Logger = l.Logger.WithFields(logrus.Fields{
		"stack": string(stack),
		"panic": fmt.Sprintf("%+v", v),
	})
}
