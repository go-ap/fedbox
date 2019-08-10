package main

import (
	"flag"
	"github.com/go-ap/activitypub/client"
	"github.com/go-ap/auth"
	"github.com/go-ap/fedbox/app"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
	"github.com/go-ap/fedbox/internal/log"
	"github.com/go-ap/fedbox/storage/boltdb"
	"github.com/go-ap/fedbox/storage/pgx"
	"github.com/go-ap/fedbox/validation"
	st "github.com/go-ap/storage"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/openshift/osin"
	"github.com/sirupsen/logrus"
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
	var oauthDB osin.Storage
	var err error
	if a.Config().Storage == config.BoltDB {
		bolt, errb := boltdb.New(boltdb.Config{
			Path:       app.Config.BoltDB(),
			BucketName: app.Config.Host,
			LogFn:      func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Infof(s, p...) },
			ErrFn:      func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Errorf(s, p...) },
		}, a.Config().BaseURL)
		repo = bolt
		err = errb
		defer bolt.Close()

		oauthDB = auth.NewBoltDBStore(auth.Config{
			Path:       app.Config.BoltDBOAuth2(),
			BucketName: app.Config.Host,
			LogFn:      func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Infof(s, p...) },
			ErrFn:      func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Errorf(s, p...) },
		})
		defer oauthDB.Close()
	}
	if a.Config().Storage == config.Postgres {
		pg, errp := pgx.New(a.Config().DB, a.Config().BaseURL, l)
		repo = pg
		err = errp
		defer pg.Close()

		oauthDB = auth.NewPgDBStore()
		defer oauthDB.Close()
	}
	if err != nil {
		l.WithField("storage", a.Config().Storage).Error(err)
	}

	osin, err := auth.NewOAuth2Server(oauthDB, l)
	if err != nil {
		l.Warn(err.Error())
	}

	r.Use(app.Repo(repo))
	r.Use(middleware.RequestID)
	r.Use(log.NewStructuredLogger(l))

	v := validation.New(a.Config().BaseURL, client.NewClient(), repo)
	r.Route("/", app.Routes(v, osin, repo.(st.ActorLoader), l))

	status := a.Run(r, wait)
	if status != 0 {
		os.Exit(status)
	}
}
