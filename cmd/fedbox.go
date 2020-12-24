package cmd

import (
	"fmt"
	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/app"
	"github.com/go-ap/fedbox/internal/env"
	"github.com/go-ap/fedbox/internal/log"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/sirupsen/logrus"
	"gopkg.in/urfave/cli.v2"
	"time"
)

const defaultTimeout = time.Second * 15

func NewApp(r chi.Router,version string) *cli.App {
	return &cli.App{
		Name:    "fedbox",
		Usage:   "fedbox instance server",
		Version: version,
		Flags: []cli.Flag{
			&cli.DurationFlag{
				Name:  "wait",
				Usage: "the duration for which the server gracefully wait for existing connections to finish",
				Value: defaultTimeout,
			},
			&cli.StringFlag{
				Name:  "env",
				Usage: fmt.Sprintf("the environment to use. Possible values: %q, %q, %q", env.DEV, env.QA, env.PROD),
				Value: "",
			},
		},
		Action: run(r, version),
	}
}

func run(r chi.Router, version string) cli.ActionFunc {
	return func(c *cli.Context) error {
		w := c.Duration("wait")
		e := c.String("env")
		l := log.New()
		a, err := app.New(l, version, e)
		if err != nil {
			l.Errorf("Unable to initialize: %s", err)
			return err
		}
		// the level enums have same values
		logger.Level = logrus.Level(a.Config().LogLevel)

		osin, err := auth.NewServer(a.OAuthStorage, l)
		if err != nil {
			l.Warn(err.Error())
			return err
		}

		r.Use(app.Repo(a.Storage))
		r.Use(middleware.RequestID)
		r.Use(log.NewStructuredLogger(l))
		r.Route("/", a.Routes(a.Config().BaseURL, osin, l))
		status := a.Run(r, w)
		if status != 0 {
			return errors.Newf("error")
		}
		return nil
	}
}
