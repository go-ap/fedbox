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

func errf(s string, par ...interface{}) {
	fmt.Printf(s, par...)
}

func main() {
	var environ string
	var typ string
	var pgRoot string
	var file string
	var reset bool

	flag.StringVar(&environ, "env", string(env.DEV), "environment")
	flag.StringVar(&typ, "type", string(config.BoltDB), "type")
	flag.StringVar(&pgRoot, "root", "postgres", "root account of postgres server")
	flag.StringVar(&file, "path", "", "path")
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
		if file == "" {
		}
		if reset {
			boltdb.Clean(conf.BoltDBPath, []byte(conf.Host))
		}
		err = boltdb.Bootstrap(conf.BoltDBPath, []byte(conf.Host), conf.BaseURL)
	}
	if config.StorageType(typ) == config.Postgres {
		// ask for root pw
		fmt.Printf("%s password: ", pgRoot)
		pgPw, _ := terminal.ReadPassword(0)
		fmt.Println()

		if file == "" {
			dir, _ := os.Getwd()
			file = path.Join(dir, "init.sql")
		}
		if reset {
			pgx.Clean(conf, pgRoot, pgPw, file)
		}
		err = pgx.Bootstrap(conf, pgRoot, pgPw, file)
	}
	if err != nil {
		errf("Unable to update %s db: %s\n", typ, err)
		os.Exit(1)
	}
	fmt.Println("OK")
}
