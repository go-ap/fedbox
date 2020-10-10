package app

import (
	"context"
	"crypto/tls"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
	"github.com/go-ap/handlers"
	st "github.com/go-ap/storage"
	"github.com/openshift/osin"
	"github.com/sirupsen/logrus"
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
	ver          string
	caches       reqCache
	Storage      st.Repository
	OAuthStorage osin.Storage
	stopFn       func()
	infFn        LogFn
	errFn        LogFn
}

var (
	emptyLogFn = func(string, ...interface{}) {}
	InfoLogFn  = func(l logrus.FieldLogger) func(logrus.Fields, string, ...interface{}) {
		return func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Infof(s, p...) }
	}
	ErrLogFn = func(l logrus.FieldLogger) func(logrus.Fields, string, ...interface{}) {
		return func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Errorf(s, p...) }
	}
)

var AnonymousAcct = account{
	username: "anonymous",
	actor:    &auth.AnonymousActor,
}

// New instantiates a new FedBOX instance
func New(l logrus.FieldLogger, ver string, environ string) (*FedBOX, error) {
	app := FedBOX{
		ver:   ver,
		caches: make(reqCache),
		infFn: emptyLogFn,
		errFn: emptyLogFn,
	}
	if l != nil {
		app.infFn = l.Infof
		app.errFn = l.Errorf
	}
	var err error
	app.conf, err = config.LoadFromEnv(env.Type(environ))
	if err == nil {
		Config = app.conf
		ap.Secure = app.conf.Secure
	}

	if err != nil {
		app.errFn("Unable to load settings from environment variables: %s", err)
		return nil, err
	} else {
		Config = app.conf
		ap.Secure = app.conf.Secure
	}
	errors.IncludeBacktrace = app.conf.Env.IsDev() || app.conf.Env.IsTest()

	db, oauth, err := Storage(app.conf, l)
	if err != nil {
		app.errFn("Unable to initialize storage backend: %s", err)
	}
	app.Storage = db
	app.OAuthStorage = oauth
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

func waitForSignal(sigChan chan os.Signal, exitChan chan int) func(LogFn) {
	return func(l LogFn) {
		for {
			s := <-sigChan
			switch s {
			case syscall.SIGHUP:
				l("SIGHUP received, reloading configuration")
				//loadEnv(a)
			// kill -SIGINT XXXX or Ctrl+c
			case syscall.SIGINT:
				l("SIGINT received, stopping")
				exitChan <- 0
			// kill -SIGTERM XXXX
			case syscall.SIGTERM:
				l("SIGITERM received, force stopping")
				exitChan <- 0
			// kill -SIGQUIT XXXX
			case syscall.SIGQUIT:
				l("SIGQUIT received, force stopping with core-dump")
				exitChan <- 0
			default:
				l("Unknown signal %d", s)
			}
		}
	}
}

// Stop
func (f *FedBOX) Stop() {
	if f.stopFn != nil {
		f.stopFn()
	}
}

// Run is the wrapper for starting the web-server and handling signals
func (f *FedBOX) Run(m http.Handler, wait time.Duration) int {
	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()

	// set local path typer to validate collections
	handlers.Typer = pathTyper{}
	// Get start/stop functions for the http server
	srvRun, srvStop := setupHttpServer(f.conf, m, ctx)
	f.stopFn = func() {
		srvStop(f.infFn, f.errFn)
		f.OAuthStorage.Close()
		f.Storage.Close()
	}
	go srvRun(f.infFn, f.errFn)

	// Add signal handlers
	sigChan := make(chan os.Signal, 1)
	exitChan := make(chan int)
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go waitForSignal(sigChan, exitChan)(f.infFn)
	code := <-exitChan

	// Doesn't block if no connections, but will otherwise wait until the timeout deadline.
	go srvStop(f.infFn, f.errFn)
	f.infFn("Shutting down")

	return code
}
