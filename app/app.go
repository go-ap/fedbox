package app

import (
	"context"
	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
	"github.com/go-ap/fedbox/storage/boltdb"
	"github.com/go-ap/fedbox/storage/pgx"
	"github.com/go-ap/handlers"
	st "github.com/go-ap/storage"
	"github.com/openshift/osin"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
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
	Storage      st.Repository
	OAuthStorage osin.Storage
	stopFn       func()
	infFn        LogFn
	errFn        LogFn
}

func getBoltStorage(c config.Options, l logrus.FieldLogger) (st.Repository, osin.Storage, error) {
	db := boltdb.New(boltdb.Config{
		Path:  c.BoltDB(),
		LogFn: func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Infof(s, p...) },
		ErrFn: func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Errorf(s, p...) },
	}, c.BaseURL)

	oauth := auth.NewBoltDBStore(auth.BoltConfig{
		Path:       c.BoltDBOAuth2(),
		BucketName: c.Host,
		LogFn:      func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Infof(s, p...) },
		ErrFn:      func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Errorf(s, p...) },
	})
	return db, oauth, nil
}
func getPgxStorage(c config.Options, l logrus.FieldLogger) (st.Repository, osin.Storage, error) {
	db, err := pgx.New(c.DB, c.BaseURL, l)

	oauth := auth.NewPgDBStore(auth.PgConfig{
		Enabled: true,
		Host:    c.DB.Host,
		Port:    c.DB.Port,
		User:    c.DB.User,
		Pw:      c.DB.Pw,
		Name:    c.DB.Name,
		LogFn:   func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Infof(s, p...) },
		ErrFn:   func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Errorf(s, p...) },
	})
	return db, oauth, err
}

func getStorage(f FedBOX, l logrus.FieldLogger) (st.Repository, osin.Storage, error) {
	if f.Config().Storage == config.BoltDB {
		return getBoltStorage(f.Config(), l)
	}
	if f.Config().Storage == config.Postgres {
		return getPgxStorage(f.Config(), l)
	}
	return nil, nil, errors.NotImplementedf("Invalid storage type %s", f.Config().Storage)
}

// New instantiates a new FedBOX instance
func New(l logrus.FieldLogger, ver string, environ string) (*FedBOX, error) {
	app := FedBOX{ver: ver}
	var err error
	if l != nil {
		app.infFn = l.Infof
		app.errFn = l.Errorf
	}
	app.conf, err = config.LoadFromEnv(env.Type(environ))
	if err == nil {
		Config = app.conf
		ap.Secure = app.conf.Secure
	}
	if err != nil {
		app.errFn("Unable to load settings from environment variables: %s", err)
		return nil, err
	}
	errors.IncludeBacktrace = app.conf.Env == env.DEV || app.conf.Env == env.TEST

	db, oauth, err := getStorage(app, l)
	app.Storage = db
	app.OAuthStorage = oauth
	return &app, err
}

func (f FedBOX) Config() config.Options {
	return f.conf
}

func (f FedBOX) listen() string {
	return f.conf.Listen
}

func setupHttpServer(listen string, m http.Handler, wait time.Duration, ctx context.Context) (func(LogFn), func(LogFn)) {
	// TODO(marius): move server run to a separate function,
	//   so we can add other tasks that can run independently.
	//   Like a queue system for lazy loading of IRIs.
	srv := &http.Server{
		Addr:         listen,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      m,
	}

	run := func(l LogFn) {
		if err := srv.ListenAndServe(); err != nil {
			if l != nil {
				l("%s", err)
			}
			os.Exit(1)
		}
	}

	stop := func(l LogFn) {
		err := srv.Shutdown(ctx)
		if err != nil && l != nil {
			l("%s", err)
		}
		select {
		case <-ctx.Done():
			if l != nil {
				l("%s", ctx.Err())
			}
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
	f.infFn("Listening on %s", f.listen())

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()

	// Get start/stop functions for the http server
	srvRun, srvStop := setupHttpServer(f.listen(), m, wait, ctx)
	f.stopFn = func() {
		srvStop(f.infFn)
		f.OAuthStorage.Close()
		f.Storage.Close()
	}
	go srvRun(f.errFn)

	// Add signal handlers
	sigChan := make(chan os.Signal, 1)
	exitChan := make(chan int)
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go waitForSignal(sigChan, exitChan)(f.infFn)
	code := <-exitChan

	// Doesn't block if no connections, but will otherwise wait until the timeout deadline.
	go srvStop(f.errFn)
	f.infFn("Shutting down")

	return code
}
