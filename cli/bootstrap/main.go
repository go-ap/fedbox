package main

import (
	"flag"
	"fmt"
	"github.com/go-ap/auth"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
	"github.com/go-ap/fedbox/storage/boltdb"
	"github.com/go-ap/fedbox/storage/pgx"
	"github.com/openshift/osin"
	"github.com/pborman/uuid"
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
			file = conf.BoltDB()
		}
		if reset {
			boltdb.Clean(file, []byte(conf.Host))
		}
		err = boltdb.Bootstrap(file, []byte(conf.Host), conf.BaseURL)
		if err != nil {
			errf("Unable to update %s db: %s\n", typ, err)
			os.Exit(1)
		}
		oauthPath := conf.BoltDBOAuth2()

		// TODO(marius): add a local Client struct that implements Client and ClientSecretMatcher interfaces with bcrypt support
		//   It could even be a struct composite from an activitystreams.Application + secret and callback properties
		//savpw, _ := bcrypt.GenerateFromPassword([]byte("yuh4ckm3?!"), 14)
		savpw := "yuh4ckm3?!"
		id := uuid.New()
		c := osin.DefaultClient{
			Id:          id,
			Secret:      savpw,
			RedirectUri: fmt.Sprintf("http://%s/oauth/callback", conf.Host),
		}
		// TODO(marius): check if there's already a client set up for current client's RedirectURI and skip adding it
		err = auth.BootstrapBoltDB(oauthPath, []byte(conf.Host), &c)
		if err != nil {
			errf("Unable to update %s db: %s\n", typ, err)
			os.Exit(1)
		}
		fmt.Printf("OAUTH_CLIENT=%s\n", id)
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
		if err != nil {
			errf("Unable to update %s db: %s\n", typ, err)
			os.Exit(1)
		}
	}
	fmt.Println("OK")
}
