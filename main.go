package main

import (
	"flag"
	"github.com/go-ap/fedbox/app"
	"github.com/go-ap/fedbox/internal/log"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/sirupsen/logrus"
	"time"
)

var version = "HEAD"

const defaultPort = 4000
const defaultTimeout = time.Second * 15

func main() {
	var wait time.Duration
	var port int
	var host string
	var env string

	flag.DurationVar(&wait, "graceful-timeout", defaultTimeout, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	flag.IntVar(&port, "port", defaultPort, "the port on which we should listen on")
	flag.StringVar(&host, "host", "", "the host on which we should listen on")
	flag.StringVar(&env, "env", "unknown", "the environment type")
	flag.Parse()

	a := app.New(host, port, env, version)
	r := chi.NewRouter()

	logger := logrus.New()
	r.Use(middleware.RequestID)
	r.Use(log.NewStructuredLogger(logger))
	r.Use(middleware.Recoverer)

	r.Route("/", app.Routes())

	a.Run(r, wait)
}
