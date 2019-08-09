package main

import (
	"flag"
	"fmt"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
	"github.com/go-ap/fedbox/storage/boltdb"
	"github.com/go-ap/fedbox/storage/pgx"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"path"
)

var version = "HEAD"

func errf(s string, par ...interface{}) {
	fmt.Printf(s, par...)
}

func main() {
	var environ string
	var typ string
	var host string
	var pgRoot string
	var dir string
	var reset bool

	flag.StringVar(&environ, "env", string(env.DEV), "environment")
	flag.StringVar(&typ, "type", string(config.BoltDB), "type")
	flag.StringVar(&host, "host", "", "host")
	flag.StringVar(&pgRoot, "root", "postgres", "root account of postgres server")
	flag.StringVar(&dir, "dir", ".", "directory")
	flag.BoolVar(&reset, "reset", false, "reset existing database")
	flag.Parse()

	var err error
	var conf config.Options

	conf, err = config.LoadFromEnv(environ)
	if err != nil {
		errf("Unable to load settings from environment variables: %s", err)
		os.Exit(1)
	}
	if config.StorageType(typ) == config.BoltDB {
		path := config.GetBoltDBPath(dir, host, env.Type(environ))
		if reset {
			boltdb.Clean(path, []byte(conf.Host))
		}
		err = boltdb.Bootstrap(path, []byte(conf.Host), conf.BaseURL)
		if err != nil {
			errf("Unable to update %s db: %s\n", typ, err)
			os.Exit(1)
		}
	}
	if config.StorageType(typ) == config.Postgres {
		// ask for root pw
		fmt.Printf("%s password: ", pgRoot)
		pgPw, _ := terminal.ReadPassword(0)
		fmt.Println()
		dir, _ := os.Getwd()
		path := path.Join(dir, "init.sql")
		if reset {
			pgx.Clean(conf, pgRoot, pgPw, path)
		}
		err = pgx.Bootstrap(conf, pgRoot, pgPw, path)
		if err != nil {
			errf("Unable to update %s db: %s\n", typ, err)
			os.Exit(1)
		}
	}
	fmt.Println("OK")
}
