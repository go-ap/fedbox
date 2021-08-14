package app

import (
	"context"
	"io"
	"syscall"

	w "git.sr.ht/~mariusor/wrapper"
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/internal/cache"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/log"
	"github.com/go-ap/handlers"
	st "github.com/go-ap/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/openshift/osin"
	"github.com/sirupsen/logrus"
)

func init() {
	// set local path typer to validate collections
	handlers.Typer = pathTyper{}
}

type LogFn func(string, ...interface{})

type fedboxStorage struct {
	repo st.Store
	oauth osin.Storage
}

func (s *fedboxStorage) Close() error {
	s.oauth.Close()
	closable, ok :=  s.repo.(io.Closer)
	if !ok {
		return nil
	}
	return closable.Close()
}

type FedBOX struct {
	R       chi.Router
	conf    config.Options
	storage fedboxStorage
	ver     string
	caches  cache.CanStore
	OAuth   authService
	stopFn  func()
	infFn   LogFn
	errFn   LogFn
}

var (
	emptyFieldsLogFn = func(logrus.Fields, string, ...interface{}) {}
	emptyLogFn       = func(string, ...interface{}) {}
	InfoLogFn        = func(l logrus.FieldLogger) func(logrus.Fields, string, ...interface{}) {
		if l == nil {
			return emptyFieldsLogFn
		}
		return func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Infof(s, p...) }
	}
	ErrLogFn = func(l logrus.FieldLogger) func(logrus.Fields, string, ...interface{}) {
		if l == nil {
			return emptyFieldsLogFn
		}
		return func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Errorf(s, p...) }
	}
)

var AnonymousAcct = account{
	username: "anonymous",
	actor:    &auth.AnonymousActor,
}

var InternalIRI = pub.IRI("https://fedbox/")

// New instantiates a new FedBOX instance
func New(l logrus.FieldLogger, ver string, conf config.Options, db st.Store, o osin.Storage) (*FedBOX, error) {
	app := FedBOX{
		ver:     ver,
		conf:    conf,
		R:       chi.NewRouter(),
		storage: fedboxStorage{repo: db, oauth: o},
		infFn:   emptyLogFn,
		errFn:   emptyLogFn,
		caches:  cache.New(conf.RequestCache),
	}
	if l != nil {
		app.infFn = l.Infof
		app.errFn = l.Errorf
	}

	errors.IncludeBacktrace = conf.Env.IsDev() || conf.Env.IsTest()

	as, err := auth.New(conf.BaseURL, app.storage.oauth, app.storage.repo, l)
	if err != nil {
		l.Warn(err.Error())
		return nil, err
	}

	app.R.Use(RepoMw(db))
	app.R.Use(middleware.RequestID)
	app.R.Use(log.NewStructuredLogger(l))

	baseIRI := pub.IRI(conf.BaseURL)
	app.OAuth = authService{
		baseIRI: baseIRI,
		auth:    as,
		genID:   GenerateID(baseIRI),
		storage: app.storage,
		logger:  l,
	}

	app.R.Group(app.Routes())
	return &app, err
}

func (f FedBOX) Config() config.Options {
	return f.conf
}

func (f *FedBOX) Storage() st.Store {
	return f.storage.repo
}

// Stop
func (f *FedBOX) Stop() {
	defer f.storage.Close()
	if f.stopFn != nil {
		f.stopFn()
	}
}

// Run is the wrapper for starting the web-server and handling signals
func (f *FedBOX) Run() error {
	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.TODO(), f.conf.TimeOut)
	defer cancel()

	listenOn := "HTTP"
	setters := []w.SetFn{w.Handler(f.R), w.ListenOn(f.conf.Listen)}
	if f.conf.Secure && len(f.conf.CertPath)+len(f.conf.KeyPath) > 0 {
		listenOn = "HTTPS"
		setters = append(setters, w.SSL(f.conf.CertPath, f.conf.KeyPath))
	}
	// Get start/stop functions for the http server
	srvRun, srvStop := w.HttpServer(ctx, setters...)
	f.infFn("Listening on %s %s", listenOn, f.conf.Listen)
	f.stopFn = func() {
		if err := srvStop(); err != nil {
			f.errFn("Err: %s", err)
		}
	}

	exit := w.RegisterSignalHandlers(w.SignalHandlers{
		syscall.SIGHUP: func(_ chan int) {
			f.infFn("SIGHUP received, reloading configuration")
		},
		syscall.SIGINT: func(exit chan int) {
			f.infFn("SIGINT received, stopping")
			exit <- 0
		},
		syscall.SIGTERM: func(exit chan int) {
			f.infFn("SIGITERM received, force stopping")
			exit <- 0
		},
		syscall.SIGQUIT: func(exit chan int) {
			f.infFn("SIGQUIT received, force stopping with core-dump")
			exit <- 0
		},
	}).Exec(func() error {
		if err := srvRun(); err != nil{
			f.errFn("Error: %s", err)
			return err
		}
		var err error
		// Doesn't block if no connections, but will otherwise wait until the timeout deadline.
		go func(e error) {
			if err = srvStop(); err != nil {
				f.errFn("Error: %s", err)
			}
		}(err)
		return err
	})
	if exit == 0 {
		f.infFn("Shutting down")
	}
	return nil
}
