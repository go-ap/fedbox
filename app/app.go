package app

import (
	"context"
	"crypto/tls"
	"git.sr.ht/~mariusor/wrapper"
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/cache"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/log"
	"github.com/go-ap/handlers"
	st "github.com/go-ap/storage"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/openshift/osin"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"syscall"
)

func actorLoader(ctx context.Context) (st.ActorLoader, bool) {
	ctxVal := ctx.Value(handlers.RepositoryKey)
	s, ok := ctxVal.(st.ActorLoader)
	return s, ok
}

func objectLoader(ctx context.Context) (st.ObjectLoader, bool) {
	ctxVal := ctx.Value(handlers.RepositoryKey)
	s, ok := ctxVal.(st.ObjectLoader)
	return s, ok
}

var Config config.Options

type LogFn func(string, ...interface{})

type FedBOX struct {
	conf         config.Options
	R            chi.Router
	ver          string
	caches       cache.CanStore
	Storage      st.Repository
	OAuthStorage osin.Storage
	stopFn       func()
	infFn        LogFn
	errFn        LogFn
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
func New(l logrus.FieldLogger, ver string, conf config.Options, db st.Repository, o osin.Storage) (*FedBOX, error) {
	app := FedBOX{
		ver:          ver,
		conf:         conf,
		R:            chi.NewRouter(),
		Storage:      db,
		OAuthStorage: o,
		infFn:        emptyLogFn,
		errFn:        emptyLogFn,
		caches:       cache.New(!(conf.Env.IsTest() || conf.Env.IsDev())),
	}
	if l != nil {
		app.infFn = l.Infof
		app.errFn = l.Errorf
	}
	Config = conf
	ap.Secure = conf.Secure
	errors.IncludeBacktrace = conf.Env.IsDev() || conf.Env.IsTest()

	osin, err := auth.NewServer(app.OAuthStorage, l)
	if err != nil {
		l.Warn(err.Error())
		return nil, err
	}

	app.R.Use(Repo(db))
	app.R.Use(middleware.RequestID)
	app.R.Use(log.NewStructuredLogger(l))
	app.R.Route("/", app.Routes(Config.BaseURL, osin, l))

	return &app, err
}

func (f FedBOX) Config() config.Options {
	return f.conf
}

func setupHttpServer(conf config.Options, m http.Handler, ctx context.Context) (func(LogFn, LogFn), func(LogFn, LogFn)) {
	// TODO(marius): move server run to a separate function, so we can add other tasks that can run independently.
	//   Like a queue system for lazy loading of IRIs.
	var serveFn func(LogFn, LogFn) error
	srv := &http.Server{
		Addr:    conf.Listen,
		Handler: m,
	}
	fileExists := func(dir string) bool {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return false
		}
		return true
	}

	if conf.Secure && fileExists(conf.CertPath) && fileExists(conf.KeyPath) {
		srv.TLSConfig = &tls.Config{
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			},
		}
		serveFn = func(infoFn LogFn, errFn LogFn) error {
			infoFn("Listening on HTTPS %s", conf.Listen)
			return srv.ListenAndServeTLS(conf.CertPath, conf.KeyPath)
		}
	} else {
		serveFn = func(infoFn LogFn, errFn LogFn) error {
			infoFn("Listening on HTTP %s", conf.Listen)
			return srv.ListenAndServe()
		}
	}

	run := func(infoFn LogFn, errFn LogFn) {
		if err := serveFn(infoFn, errFn); err != nil {
			errFn("%s", err)
			os.Exit(1)
		}
	}

	stop := func(infoFn LogFn, errFn LogFn) {
		err := srv.Shutdown(ctx)
		if err != nil {
			errFn("%s", err)
		}
		select {
		case <-ctx.Done():
			errFn("%s", ctx.Err())
		}
	}
	// Run our server in a goroutine so that it doesn't block.
	return run, stop
}


// Stop
func (f *FedBOX) Stop() {
	if f.stopFn != nil {
		f.stopFn()
	}
}

// Run is the wrapper for starting the web-server and handling signals
func (f *FedBOX) Run() error {
	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.TODO(), f.conf.TimeOut)
	defer cancel()

	// set local path typer to validate collections
	handlers.Typer = pathTyper{}
	// Get start/stop functions for the http server
	srvRun, srvStop := setupHttpServer(f.conf, f.R, ctx)
	f.stopFn = func() {
		srvStop(f.infFn, f.errFn)
		f.OAuthStorage.Close()
		f.Storage.Close()
	}

	wrapper.RegisterSignalHandlers(wrapper.SignalHandlers{
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
	}).Exec(func() {
		srvRun(f.infFn, f.errFn)
		// Doesn't block if no connections, but will otherwise wait until the timeout deadline.
		go srvStop(f.infFn, f.errFn)
		f.infFn("Shutting down")
	})
	return nil
}
