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

func NewApp(r chi.Router, l logrus.FieldLogger, version string) cli.App {
	a, _ := app.New(l, version, "dev")
	return cli.App{
		Name:    "fedbox",
		Usage:   "fedbox instance server",
		Version: version,
		Before:  beforeApp(a, r, l, version),
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
		Action: run(a, r),
	}
}

func beforeApp(a *app.FedBOX, r chi.Router, l logrus.FieldLogger, version string) cli.BeforeFunc {
	return func(c *cli.Context) error {
		osin, err := auth.NewOAuth2Server(a.OAuthStorage, l)
		if err != nil {
			l.Warn(err.Error())
			return err
		}

		r.Use(app.Repo(a.Storage))
		r.Use(middleware.RequestID)
		r.Use(log.NewStructuredLogger(l))
		r.Route("/", a.Routes(a.Config().BaseURL, osin, l))
		return nil
	}
}

func run(a *app.FedBOX, r chi.Router) cli.ActionFunc {
	return func(c *cli.Context) error {
		wait := c.Duration("wait")
		status := a.Run(r, wait)
		if status != 0 {
			return errors.Newf("error")
		}
		return nil
	}
}
