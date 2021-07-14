package cmd

import (
	"fmt"
	"github.com/go-ap/fedbox/app"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
	"github.com/go-ap/fedbox/internal/log"
	"gopkg.in/urfave/cli.v2"
	"net/http/pprof"
	"time"
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
		},
		Action: run(version),
	}
}

func run(version string) cli.ActionFunc {
	return func(c *cli.Context) error {
		w := c.Duration("wait")
		e := c.String("env")
		conf, err := config.LoadFromEnv(env.Type(e), w)
		if err != nil {
			return err
		}
		l := log.New(conf.LogLevel)
		db, o, err := app.Storage(conf, l)
		if err != nil {
			l.Errorf("Unable to initialize storage backend: %s", err)
		}
		a, err := app.New(l, version, conf, db, o)
		if err != nil {
			l.Errorf("Unable to initialize: %s", err)
			return err
		}

		if conf.Env.IsDev() {
			// Register pprof handlers
			a.R.HandleFunc("/debug/pprof/", pprof.Index)
			a.R.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
			a.R.HandleFunc("/debug/pprof/profile", pprof.Profile)
			a.R.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
			a.R.HandleFunc("/debug/pprof/trace", pprof.Trace)
		}

		return a.Run()
	}
}
