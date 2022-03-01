package cmd

import (
	"fmt"
	"os"

	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/config"
	s "github.com/go-ap/fedbox/storage"
	"github.com/urfave/cli/v2"
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
		service := ap.Self(ap.DefaultServiceIRI(c.Conf.BaseURL))
		if _, err := c.Storage.Save(&service); err != nil {
			Errf("Error adding service: %s\n", err)
		}
		if metaSaver, ok := ctl.Storage.(s.MetadataTyper); ok {
			if err := AddKeyToItem(metaSaver, &service); err != nil {
				Errf("Error saving metadata for service: %s", err)
			}
		}
		return Bootstrap(c.Conf)
	}
}

func Bootstrap(conf config.Options) error {
	if err := bootstrapFn(conf); err != nil {
		return errors.Annotatef(err, "Unable to create %s db for storage %s", conf.BaseStoragePath(), conf.Storage)
	}
	fmt.Fprintf(os.Stdout, "Successfuly created %s db for storage %s\n", conf.BaseStoragePath(), conf.Storage)
	return nil
}

func Reset(conf config.Options) error {
	if err := cleanFn(conf); err != nil {
		return errors.Annotatef(err, "Unable to reset %s db for storage %s", conf.BaseStoragePath(), conf.Storage)
	}
	fmt.Fprintf(os.Stdout, "Successful reset %s db for storage %s\n", conf.BaseStoragePath(), conf.Storage)
	return nil
}
