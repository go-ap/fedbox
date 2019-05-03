package app

import (
	"context"
	"fmt"
	"github.com/go-ap/fedbox/internal/env"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Fedbox struct {
	host string
	port int
	env env.Type
}

// New instantiates a new Fedbox
func New(host string, port int, environ string, ver string) Fedbox {
	app := Fedbox{host: host, port: port}
	if !env.ValidType(environ) {
		app.env = env.DEV
	} else {
		app.env = env.Type(environ)
	}
	return app
}

func (a Fedbox) listen() string {
	return fmt.Sprintf("%s:%d", a.host, a.port)
}

// Run is the wrapper for starting the web-server and handling signals
func (a *Fedbox) Run(m http.Handler, wait time.Duration) {
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
			//a.Logger.Error(err.Error())
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
				//a.Logger.Info("SIGHUP received, reloading configuration")
				//loadEnv(a)
			// kill -SIGINT XXXX or Ctrl+c
			case syscall.SIGINT:
				//a.Logger.Info("SIGINT received, stopping")
				exitChan <- 0
			// kill -SIGTERM XXXX
			case syscall.SIGTERM:
				//a.Logger.Info("SIGITERM received, force stopping")
				exitChan <- 0
			// kill -SIGQUIT XXXX
			case syscall.SIGQUIT:
				//a.Logger.Info("SIGQUIT received, force stopping with core-dump")
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
	//a.Logger.Info("Shutting down")
	os.Exit(code)
}
