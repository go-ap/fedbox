package main

import (
	"flag"
	"github.com/go-ap/fedbox/app"
	"github.com/go-ap/fedbox/internal/log"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"time"
)

var version = "HEAD"

const defaultPort = 4000
const defaultTimeout = time.Second * 15

func main() {
	var wait time.Duration
	var port int

	flag.DurationVar(&wait, "graceful-timeout", defaultTimeout, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	flag.IntVar(&port, "port", defaultPort, "the port on which we should listen on")
	flag.Parse()

	l := log.New()
	a := app.New(port, l, version)
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(log.NewStructuredLogger(l))
	r.Use(middleware.Recoverer)

	r.Route("/", app.Routes())

	a.Run(r, wait)
}
