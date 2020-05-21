package cmd

import (
	"fmt"
	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
	"github.com/go-ap/fedbox/storage/badger"
	"github.com/go-ap/fedbox/storage/boltdb"
	"github.com/go-ap/fedbox/storage/pgx"
	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/urfave/cli.v2"
	"os"
	"path"
)

var Bootstrap = &cli.Command{
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

func getConfig(c *cli.Context) (string, config.StorageType, env.Type) {
	// @todo(marius): move this to a Before function
	dir := c.String("dir")
	if dir == "" {
		dir = "."
	}
	environ := env.Type(c.String("env"))
	if environ == "" {
		environ = env.DEV
	}
	typ := config.StorageType(c.String("type"))
	if typ == "" {
		typ = config.BoltDB
	}
	if opt, err := config.LoadFromEnv(env.Type(typ)); err == nil {
		if opt.StoragePath != os.TempDir() {
			dir = opt.StoragePath
		}
	}
	return dir, typ, environ
}

func resetAct(c *Control) cli.ActionFunc {
	return func(c *cli.Context) error {
		dir, typ, environ := getConfig(c)
		err := ctl.BootstrapReset(dir, typ, environ)
		if err != nil {
			return err
		}
		return ctl.Bootstrap(dir, typ, environ)
	}
}

func bootstrapAct(c *Control) cli.ActionFunc {
	return func(c *cli.Context) error {
		dir, typ, environ := getConfig(c)
		return ctl.Bootstrap(dir, typ, environ)
	}
}

func (c *Control) Bootstrap(dir string, typ config.StorageType, environ env.Type) error {
	if typ == config.BoltDB {
		storagePath := config.GetDBPath(dir, c.Host, environ)
		err := boltdb.Bootstrap(storagePath, c.BaseURL)
		if err != nil {
			return errors.Annotatef(err, "Unable to create %s db", storagePath)
		}
		oauthPath := config.GetDBPath(dir, fmt.Sprintf("%s-oauth", c.Host), environ)
		if _, err := os.Stat(oauthPath); os.IsNotExist(err) {
			err := auth.BootstrapBoltDB(oauthPath, []byte(c.Host))
			if err != nil {
				return errors.Annotatef(err, "Unable to create %s db", oauthPath)
			}
		}
	}
	if typ == config.Badger {
		storagePath, err := badger.Path(c.Conf)
		if err != nil {
			return err
		}
		err = badger.Bootstrap(storagePath, c.Conf.BaseURL)
		if err != nil {
			return err
		}
	}
	var pgRoot string
	if typ == config.Postgres {
		// ask for root pw
		fmt.Printf("%s password: ", pgRoot)
		pgPw, _ := terminal.ReadPassword(0)
		fmt.Println()
		dir, _ := os.Getwd()
		path := path.Join(dir, "init.sql")
		err := pgx.Bootstrap(c.Conf, pgRoot, pgPw, path)
		if err != nil {
			return errors.Annotatef(err, "Unable to update %s db", typ)
		}
	}
	return nil
}

func (c *Control) BootstrapReset(dir string, typ config.StorageType, environ env.Type) error {
	if typ == config.BoltDB {
		path := config.GetDBPath(dir, c.Host, environ)
		err := boltdb.Clean(path)
		if err != nil {
			return errors.Annotatef(err, "Unable to update %s db", typ)
		}
	}
	var pgRoot string
	if typ == config.Postgres {
		// ask for root pw
		fmt.Printf("%s password: ", pgRoot)
		pgPw, _ := terminal.ReadPassword(0)
		fmt.Println()
		dir, _ := os.Getwd()
		path := path.Join(dir, "init.sql")
		err := pgx.Clean(c.Conf, pgRoot, pgPw, path)
		if err != nil {
			return errors.Annotatef(err, "Unable to update %s db", typ)
		}
	}
	if typ == config.Badger {
		path, err := badger.Path(c.Conf)
		if err != nil {
			return fmt.Errorf("unable to update %s db: %w", typ, err)
		}
		err = badger.Clean(path)
		if err != nil {
			return fmt.Errorf("unable to update %s db: %w", typ, err)
		}
	}
	return nil
}
