package log

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"git.sr.ht/~mariusor/lw"
	"github.com/go-chi/chi/v5/middleware"
)

type Conf struct {
	Output io.Writer
	Type   string
	Pretty bool
	Level  lw.Level
}

func New(conf Conf) lw.Logger {
	if conf.Output == nil {
		conf.Output = os.Stdout
	}
	l := lw.Dev(lw.SetLevel(conf.Level), lw.SetOutput(conf.Output))
	return l
}

func NewStructuredLogger(logger lw.Logger) func(next http.Handler) http.Handler {
	return middleware.RequestLogger(&StructuredLogger{logger})
}

type StructuredLogger struct {
	Logger lw.Logger
}

func (l *StructuredLogger) NewLogEntry(r *http.Request) middleware.LogEntry {
	entry := &StructuredLoggerEntry{}

	entry.Logger = l.Logger
	logFields := lw.Ctx{}

	logFields["ts"] = time.Now().UTC().Format(time.RFC1123)

	if reqID := middleware.GetReqID(r.Context()); reqID != "" {
		logFields["req_id"] = reqID
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

	entry.Logger = entry.Logger.WithContext(logFields)

	return entry
}

type StructuredLoggerEntry struct {
	Logger lw.Logger
}

func (l *StructuredLoggerEntry) Write(status, bytes int, header http.Header, elapsed time.Duration, extra interface{}) {
	l.Logger = l.Logger.WithContext(lw.Ctx{
		"status":  status,
		"len":     bytes,
		"elapsed": elapsed,
	})

	var logFn func(string, ...interface{})
	if status >= 200 && status < 400 {
		logFn = l.Logger.Infof
	} else {
		logFn = l.Logger.Errorf
	}
	logFn(http.StatusText(status))
}

func (l *StructuredLoggerEntry) Panic(v interface{}, stack []byte) {
	l.Logger = l.Logger.WithContext(lw.Ctx{
		"stack": string(stack),
		"panic": fmt.Sprintf("%+v", v),
	})
}
