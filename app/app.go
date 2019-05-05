package app

import (
	"context"
	"fmt"
	as "github.com/go-ap/activitystreams"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
	"github.com/go-ap/fedbox/internal/errors"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type logFn func(string, ...interface{})

type Fedbox struct {
	conf config.Options
	ver  string
	port int
	warn logFn
	err  logFn
	inf  logFn
	dbg  logFn
}

// New instantiates a new Fedbox instance
func New(port int, l logrus.FieldLogger, ver string) Fedbox {
	app := Fedbox{port: port, ver: ver}
	var err error
	if l != nil {
		app.dbg = l.Debugf
		app.inf = l.Infof
		app.warn = l.Warnf
		app.err = l.Errorf
	}
	app.conf, err = config.LoadFromEnv()
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
	return fmt.Sprintf("%s:%d", a.conf.Host, a.port)
}

// Run is the wrapper for starting the web-server and handling signals
func (a *Fedbox) Run(m http.Handler, wait time.Duration) {
	a.inf("Started")
	srv := &http.Server{
		Addr:         a.listen(),
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      m,
	}

	// Run our server in a goroutine so that it doesn't block.
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			a.err("%s", err.Error())
			os.Exit(1)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGINT,
		syscall.SIGTERM, syscall.SIGQUIT)

	exitChan := make(chan int)
	go func() {
		for {
			s := <-sigChan
			switch s {
			case syscall.SIGHUP:
				a.inf("SIGHUP received, reloading configuration")
				//loadEnv(a)
			// kill -SIGINT XXXX or Ctrl+c
			case syscall.SIGINT:
				a.inf("SIGINT received, stopping")
				exitChan <- 0
			// kill -SIGTERM XXXX
			case syscall.SIGTERM:
				a.inf("SIGITERM received, force stopping")
				exitChan <- 0
			// kill -SIGQUIT XXXX
			case syscall.SIGQUIT:
				a.inf("SIGQUIT received, force stopping with core-dump")
				exitChan <- 0
			default:
				//a.Logger.WithContext(log.Ctx{"signal": s}).Info("Unknown signal")
			}
		}
	}()
	code := <-exitChan

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	//log.RegisterExitHandler(cancel)
	defer cancel()

	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	srv.Shutdown(ctx)
	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	a.inf("Shutting down")
	os.Exit(code)
}
