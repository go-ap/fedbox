package cmd

import (
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/internal/config"
	"gopkg.in/urfave/cli.v2"
)

var BootstrapCmd = &cli.Command{
	Name:  "bootstrap",
	Usage: "Bootstrap a new postgres or bolt database helper",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "root",
			Usage: "root account of postgres server (default: postgres)",
			Value: "postgres",
		},
		&cli.StringFlag{
			Name:  "sql",
			Usage: "path to the queries for initializing the database",
			Value: "postgres",
		},
	},
	Action: bootstrapAct(&ctl),
	Subcommands: []*cli.Command{
		reset,
	},
}

var reset = &cli.Command{
	Name:   "reset",
	Usage:  "reset an existing database",
	Action: resetAct(&ctl),
}

func resetAct(c *Control) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		err := Reset(c.Conf)
		if err != nil {
			return err
		}
		return Bootstrap(c.Conf)
	}
}

func bootstrapAct(c *Control) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		return Bootstrap(c.Conf)
	}
}

func Bootstrap(conf config.Options) error {
	if err := bootstrapFn(conf); err != nil {
		return errors.Annotatef(err, "Unable to create %s db for storage %s", conf.StoragePath, conf.Storage)
	}
	return nil
}

func Reset(conf config.Options) error {
	return cleanFn(conf)
}
