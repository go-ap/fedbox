package cmd

import (
	"fmt"
	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/storage/badger"
	"github.com/go-ap/fedbox/storage/boltdb"
	"github.com/go-ap/fedbox/storage/fs"
	"github.com/go-ap/fedbox/storage/pgx"
	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/urfave/cli.v2"
	"os"
	"path"
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
		err := bootstrapReset(c.Conf)
		if err != nil {
			return err
		}
		return bootstrap(c.Conf)
	}
}

func bootstrapAct(c *Control) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		return bootstrap(c.Conf)
	}
}

func bootstrap(conf config.Options) error {
	if conf.Storage == config.BoltDB {
		err := boltdb.Bootstrap(conf)
		if err != nil {
			return errors.Annotatef(err, "Unable to create %s db", conf.StoragePath)
		}
		oauthPath := config.GetDBPath(conf.StoragePath, fmt.Sprintf("%s-oauth", conf.Host), conf.Env)
		if _, err := os.Stat(oauthPath); os.IsNotExist(err) {
			err := auth.BootstrapBoltDB(oauthPath, []byte(conf.Host))
			if err != nil {
				return errors.Annotatef(err, "Unable to create %s db", oauthPath)
			}
		}
	}
	if conf.Storage == config.Badger {
		return badger.Bootstrap(conf)
	}
	if conf.Storage == config.FS {
		return fs.Bootstrap(conf)
	}
	var pgRoot string
	if conf.Storage == config.Postgres {
		// ask for root pw
		fmt.Printf("%s password: ", pgRoot)
		pgPw, _ := terminal.ReadPassword(0)
		fmt.Println()
		dir, _ := os.Getwd()
		path := path.Join(dir, "init.sql")
		err := pgx.Bootstrap(conf, pgRoot, pgPw, path)
		if err != nil {
			return errors.Annotatef(err, "Unable to update %s db", conf.Storage)
		}
	}
	return nil
}

func bootstrapReset(conf config.Options) error {
	if conf.Storage == config.BoltDB {
		return boltdb.Clean(conf)
	}
	if conf.Storage == config.Postgres {
		var pgRoot string
		// ask for root pw
		fmt.Printf("%s password: ", pgRoot)
		pgPw, _ := terminal.ReadPassword(0)
		fmt.Println()
		dir, _ := os.Getwd()
		path := path.Join(dir, "init.sql")
		err := pgx.Clean(conf, pgRoot, pgPw, path)
		if err != nil {
			return errors.Annotatef(err, "Unable to update %s db", conf.Storage)
		}
	}
	if conf.Storage == config.Badger {
		return badger.Clean(conf)
	}
	return nil
}
