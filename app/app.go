package app

import (
	"context"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
	"github.com/go-ap/handlers"
	st "github.com/go-ap/storage"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)
type CtxtKey string

var actorKey = CtxtKey("__actor")

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

// TODO(marius) Move away from keeping this data statically.
var Config config.Options

type logFn func(string, ...interface{})

type runner func()

type Fedbox struct {
	conf config.Options
	ver  string
	warn logFn
	err  logFn
	inf  logFn
	dbg  logFn
}

// New instantiates a new Fedbox instance
func New(l logrus.FieldLogger, ver string) Fedbox {
	app := Fedbox{ver: ver}
	var err error
	if l != nil {
		app.dbg = l.Debugf
		app.inf = l.Infof
		app.warn = l.Warnf
		app.err = l.Errorf
	}
	app.conf, err = config.LoadFromEnv()
	if err == nil {
		Config = app.conf
		ap.Secure = app.conf.Secure
	}
	if err != nil {
		app.warn("Unable to load settings from environment variables: %s", err)
	}
	errors.IncludeBacktrace = app.conf.Env == env.DEV
	as.ItemTyperFunc = ap.JSONGetItemByType
	return app
}

func (a Fedbox) Config() config.Options {
	return a.conf
}

func (a Fedbox) listen() string {
	return a.conf.Listen
}

func setupHttpServer(listen string, m http.Handler, wait time.Duration, ctx context.Context) (func(logFn), func(logFn)) {
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

	run := func(l logFn) {
		if err := srv.ListenAndServe(); err != nil {
			l("%s", err)
			os.Exit(1)
		}
	}

	stop := func(l logFn) {
		err := srv.Shutdown(ctx)
		if err != nil {
			l("%s", err)
		}
		select {
		case <-ctx.Done():
			l("%s", ctx.Err())
		}
	}
	// Run our server in a goroutine so that it doesn't block.
	return run, stop
}

func waitForSignal(sigChan chan os.Signal, exitChan chan int) func(logFn) {
	return func(l logFn) {
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

// Run is the wrapper for starting the web-server and handling signals
func (a *Fedbox) Run(m http.Handler, wait time.Duration) int {
	a.inf("Listening on %s", a.listen())

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()

	// Get start/stop functions for the http server
	srvRun, srvStop := setupHttpServer(a.listen(), m, wait, ctx)
	go srvRun(a.err)

	// Add signal handlers
	sigChan := make(chan os.Signal, 1)
	exitChan := make(chan int)
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go waitForSignal(sigChan, exitChan)(a.inf)
	code := <-exitChan

	// Doesn't block if no connections, but will otherwise wait until the timeout deadline.
	go srvStop(a.err)
	a.inf("Shutting down")

	return code
}
