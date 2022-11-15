package fedbox

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"git.sr.ht/~mariusor/lw"
	w "git.sr.ht/~mariusor/wrapper"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/cache"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
	st "github.com/go-ap/fedbox/storage"
	"github.com/go-ap/processing"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/openshift/osin"
)

func init() {
	// set local path typer to validate collections
	processing.Typer = pathTyper{}
}

type LogFn func(string, ...interface{})

type fedboxStorage struct {
	repo  processing.Store
	oauth osin.Storage
}

func (s *fedboxStorage) Close() error {
	s.oauth.Close()
	closable, ok := s.repo.(io.Closer)
	if !ok {
		return nil
	}
	return closable.Close()
}

type FedBOX struct {
	R       chi.Router
	conf    config.Options
	self    vocab.Service
	storage fedboxStorage
	ver     string
	caches  cache.CanStore
	OAuth   authService
	stopFn  func()
	infFn   LogFn
	errFn   LogFn
}

var (
	emptyFieldsLogFn = func(lw.Ctx, string, ...interface{}) {}
	emptyLogFn       = func(string, ...interface{}) {}
	emptyStopFn      = func() {}
	InfoLogFn        = func(l lw.Logger) func(lw.Ctx, string, ...interface{}) {
		if l == nil {
			return emptyFieldsLogFn
		}
		return func(f lw.Ctx, s string, p ...interface{}) { l.WithContext(f).Infof(s, p...) }
	}
	ErrLogFn = func(l lw.Logger) func(lw.Ctx, string, ...interface{}) {
		if l == nil {
			return emptyFieldsLogFn
		}
		return func(f lw.Ctx, s string, p ...interface{}) { l.WithContext(f).Errorf(s, p...) }
	}
)

var AnonymousAcct = account{
	username: "anonymous",
	actor:    &auth.AnonymousActor,
}

var InternalIRI = vocab.IRI("https://fedbox/")

func Config(e string, to time.Duration) (config.Options, error) {
	return config.LoadFromEnv(env.Type(e), to)
}

// New instantiates a new FedBOX instance
func New(l lw.Logger, ver string, conf config.Options, db processing.Store, o osin.Storage) (*FedBOX, error) {
	if db == nil {
		return nil, errors.Newf("invalid storage")
	}
	app := FedBOX{
		ver:     ver,
		conf:    conf,
		R:       chi.NewRouter(),
		storage: fedboxStorage{repo: db, oauth: o},
		stopFn:  emptyStopFn,
		infFn:   emptyLogFn,
		errFn:   emptyLogFn,
		caches:  cache.New(conf.RequestCache),
	}
	if l != nil {
		app.infFn = l.Infof
		app.errFn = l.Errorf
	}

	errors.IncludeBacktrace = conf.LogLevel == lw.TraceLevel

	selfIRI := ap.DefaultServiceIRI(conf.BaseURL)
	self, _ := db.Load(selfIRI)
	vocab.OnActor(self, func(actor *vocab.Actor) error {
		app.self = *actor
		return nil
	})
	if app.self.ID != selfIRI {
		app.infFn("trying to bootstrap the instance's self service")
		if saver, ok := db.(st.CanBootstrap); ok {
			app.self = ap.Self(selfIRI)
			if err := saver.CreateService(app.self); err != nil {
				app.errFn("unable to save the instance's self service: %s", err)
				return nil, err
			}
		}
	}

	as, err := auth.New(conf.BaseURL, app.storage.oauth, app.storage.repo, l)
	if err != nil {
		l.Warnf(err.Error())
		return nil, err
	}

	app.R.Use(middleware.RequestID)
	app.R.Use(lw.Middlewares(l)...)

	baseIRI := app.self.GetLink()
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

func (f FedBOX) Storage() processing.Store {
	return f.storage.repo
}

// Stop
func (f *FedBOX) Stop() {
	f.storage.Close()
	f.stopFn()
}

func (f *FedBOX) reload() (err error) {
	f.conf, err = config.LoadFromEnv(f.conf.Env, f.conf.TimeOut)
	f.caches.Remove()
	return err
}

func (f FedBOX) actorFromRequest(r *http.Request) *vocab.Actor {
	act, err := f.OAuth.auth.LoadActorFromAuthHeader(r)
	if err != nil {
		return nil
	}
	return act
}

// Run is the wrapper for starting the web-server and handling signals
func (f *FedBOX) Run(c context.Context) error {
	// Create a deadline to wait for.
	ctx, cancelFn := context.WithTimeout(c, f.conf.TimeOut)
	defer cancelFn()

	listenOn := ""
	setters := []w.SetFn{w.Handler(f.R)}
	dir, _ := filepath.Split(f.conf.Listen)
	if _, err := os.Stat(dir); err == nil {
		listenOn = "socket"
		setters = append(setters, w.Socket(f.conf.Listen))
		defer func() { os.RemoveAll(f.conf.Listen) }()
	} else {
		if f.conf.Secure && len(f.conf.CertPath)+len(f.conf.KeyPath) > 0 {
			listenOn = "HTTPS"
			setters = append(setters, w.HTTPS(f.conf.Listen, f.conf.CertPath, f.conf.KeyPath))
		} else {
			listenOn = "HTTP"
			setters = append(setters, w.HTTP(f.conf.Listen))
		}
	}
	// Get start/stop functions for the http server
	srvRun, srvStop := w.HttpServer(setters...)
	f.infFn("Started %s %s %s", f.conf.BaseURL, listenOn, f.conf.Listen)
	f.stopFn = func() {
		if err := srvStop(ctx); err != nil {
			f.errFn("Err: %s", err)
		}
	}

	exit := w.RegisterSignalHandlers(w.SignalHandlers{
		syscall.SIGHUP: func(_ chan int) {
			f.infFn("SIGHUP received, reloading configuration")
			if err := f.reload(); err != nil {
				f.errFn("Failed: %s", err.Error())
			}
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
		if err := srvRun(); err != nil {
			f.errFn("Error: %s", err)
			return err
		}
		var err error
		// Doesn't block if no connections, but will otherwise wait until the timeout deadline.
		go func(e error) {
			f.errFn("Error: %s", err)
			f.stopFn()
		}(err)
		return err
	})
	if exit == 0 {
		f.infFn("Shutting down")
	}
	return nil
}
