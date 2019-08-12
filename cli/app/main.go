package main

import (
	"fmt"
	"github.com/go-ap/activitypub/client"
	"github.com/go-ap/activitystreams"
	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/activitypub"
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
	"gopkg.in/urfave/cli.v2"
	"os"
	"time"
)

var version = "HEAD"

const defaultTimeout = time.Second * 15

func main() {
	activitystreams.ItemTyperFunc = activitypub.ItemByType

	var a *app.FedBOX
	var repo st.Loader
	var oauthDB osin.Storage

	var wait time.Duration

	r := chi.NewRouter()
	l := log.New()

	srv := cli.App{}
	srv.Name = "fedbox"
	srv.Usage = "fedbox instance server"
	srv.Version = version
	srv.Before = func(c *cli.Context) error {
		environ := env.Type(c.String("env"))
		if environ == "" {
			environ = env.DEV
		}

		a = app.New(l, version, environ)
		if a.Config().Storage == config.BoltDB {
			bolt := boltdb.New(boltdb.Config{
				Path:       app.Config.BoltDB(),
				BucketName: app.Config.Host,
				LogFn:      func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Infof(s, p...) },
				ErrFn:      func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Errorf(s, p...) },
			}, a.Config().BaseURL)
			defer bolt.Close()
			repo = bolt

			oauthDB = auth.NewBoltDBStore(auth.Config{
				Path:       app.Config.BoltDBOAuth2(),
				BucketName: app.Config.Host,
				LogFn:      func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Infof(s, p...) },
				ErrFn:      func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Errorf(s, p...) },
			})
			defer oauthDB.Close()
		}
		var err error

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
			return err
		}

		osin, err := auth.NewOAuth2Server(oauthDB, l)
		if err != nil {
			l.Warn(err.Error())
			return err
		}

		r.Use(app.Repo(repo))
		r.Use(middleware.RequestID)
		r.Use(log.NewStructuredLogger(l))

		v := validation.New(a.Config().BaseURL, client.NewClient(), repo)
		r.Route("/", app.Routes(v, osin, repo.(st.ActorLoader), l))
		return nil
	}
	srv.Flags = []cli.Flag{
		&cli.DurationFlag{
			Name:        "wait",
			Usage:       "the duration for which the server gracefully wait for existing connections to finish",
			Value:       defaultTimeout,
			Destination: &wait,
		},
		&cli.StringFlag{
			Name:  "env",
			Usage: fmt.Sprintf("the environment to use. Possible values: %q, %q, %q", env.DEV, env.QA, env.PROD),
			Value: string(env.DEV),
		},
	}
	srv.Action = func(c *cli.Context) error {
		status := a.Run(r, wait)
		if status != 0 {
			return errors.Newf("error")
		}
		return nil
	}

	err := srv.Run(os.Args)
	if err != nil {
		l.Errorf("Error: %s", err)
		os.Exit(1)
	}
}
