package main

import (
	"fmt"
	"github.com/go-ap/activitypub/client"
	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/app"
	"github.com/go-ap/fedbox/internal/env"
	"github.com/go-ap/fedbox/internal/log"
	"github.com/go-ap/fedbox/validation"
	st "github.com/go-ap/storage"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"gopkg.in/urfave/cli.v2"
	"os"
	"time"
)

var version = "HEAD"

const defaultTimeout = time.Second * 15

func main() {
	var a *app.FedBOX
	var wait time.Duration

	r := chi.NewRouter()
	l := log.New()

	srv := cli.App{}
	srv.Name = "fedbox"
	srv.Usage = "fedbox instance server"
	srv.Version = version
	srv.Before = func(c *cli.Context) error {
		environ := c.String("env")

		var err error
		a, err = app.New(l, version, environ)
		if err != nil {
			l.WithField("storage", a.Config().Storage).Error(err)
			return err
		}

		osin, err := auth.NewOAuth2Server(a.OAuthStorage, l)
		if err != nil {
			l.Warn(err.Error())
			return err
		}

		r.Use(app.Repo(a.Storage))
		r.Use(middleware.RequestID)
		r.Use(log.NewStructuredLogger(l))

		v := validation.New(a.Config().BaseURL, client.NewClient(), a.Storage)
		r.Route("/", app.Routes(a.Config().BaseURL, v, osin, a.Storage.(st.ActorLoader), l))
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
			Value: "",
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
