package main

import (
	"flag"
	"github.com/go-ap/fedbox/app"
	"github.com/go-ap/fedbox/internal/log"
	"github.com/go-ap/fedbox/storage"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/jackc/pgx"
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

	dbConf := a.Config().DB
	conn, err := pgx.Connect(pgx.ConnConfig{
		Host: dbConf.Host,
		Port: uint16(dbConf.Port),
		Database: dbConf.Name,
		User: dbConf.User,
		Password: dbConf.Pw,
		Logger: storage.DBLogger(l),
	})
	defer conn.Close()
	if err == nil {
		r.Use(app.Repo(storage.New(conn)))
	} else {
		l.Errorf("invalid db connection")
	}


	r.Use(middleware.RequestID)
	r.Use(log.NewStructuredLogger(l))


	r.Route("/", app.Routes())

	a.Run(r, wait)
}
