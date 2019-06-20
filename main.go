package main

import (
	"flag"
	"github.com/go-ap/fedbox/app"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
	"github.com/go-ap/fedbox/internal/log"
	"github.com/go-ap/fedbox/storage/boltdb"
	"github.com/go-ap/fedbox/storage/pgx"
	st "github.com/go-ap/storage"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"os"
	"time"
)

var version = "HEAD"

const defaultTimeout = time.Second * 15

func main() {
	var wait time.Duration
	var environ string

	flag.DurationVar(&wait, "graceful-timeout", defaultTimeout, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	flag.StringVar(&environ, "env", string(env.DEV), "environment")
	flag.Parse()

	l := log.New()
	a := app.New(l, version, environ)
	r := chi.NewRouter()

	var repo st.Loader
	var err error
	if a.Config().Storage == config.BoltDB {
		bolt, errb := boltdb.New(boltdb.Config{
			Path:       app.Config.BoltDBPath,
			BucketName: app.Config.Host,
		}, a.Config().BaseURL)
		repo = bolt
		err = errb
		defer bolt.Close()
	}
	if a.Config().Storage == config.Postgres {
		pg, errp := pgx.New(a.Config().DB, a.Config().BaseURL, l)
		repo = pg
		err = errp
		defer pg.Close()
	}
	if err != nil {
		l.WithField("storage", a.Config().Storage).Error(err)
	}

	r.Use(app.Repo(repo))
	r.Use(middleware.RequestID)
	r.Use(log.NewStructuredLogger(l))

	r.Route("/", app.Routes())

	status := a.Run(r, wait)
	os.Exit(status)
}
