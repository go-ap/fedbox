package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/go-ap/fedbox/app"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
	"github.com/go-ap/fedbox/internal/log"
	"github.com/urfave/cli/v2"
)

const defaultTimeout = time.Second * 15

func NewApp(version string) *cli.App {
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
			&cli.BoolFlag{
				Name:   "profile",
				Hidden: true,
				Value:  false,
			},
		},
		Action: run(version),
	}
}

func run(version string) cli.ActionFunc {
	return func(c *cli.Context) error {
		w := c.Duration("wait")
		e := c.String("env")

		conf, err := config.LoadFromEnv(env.Type(e), w)
		conf.Profile = c.Bool("profile")
		conf.Secure = conf.Secure && !conf.Profile

		if err != nil {
			return err
		}
		l := log.New(log.Conf{Type: conf.LogOutput, Pretty: !conf.Env.IsProd(), Level: conf.LogLevel})
		db, o, err := app.Storage(conf, l)
		if err != nil {
			l.Errorf("Unable to initialize storage backend: %s", err)
		}
		a, err := app.New(l, version, conf, db, o)
		if err != nil {
			l.Errorf("Unable to initialize: %s", err)
			return err
		}

		return a.Run(context.Background())
	}
}
