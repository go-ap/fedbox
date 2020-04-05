package cmd

import (
	"bytes"
	"fmt"
	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/app"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
	"github.com/go-ap/fedbox/storage/badger"
	"github.com/go-ap/fedbox/storage/boltdb"
	"github.com/go-ap/fedbox/storage/pgx"
	"github.com/go-ap/storage"
	"github.com/openshift/osin"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/urfave/cli.v2"
	"os"
)

type Control struct {
	Conf        config.Options
	Host        string
	BaseURL     string
	AuthStorage osin.Storage
	Storage     storage.Repository
}

func New(authDB osin.Storage, actorDb storage.Repository, conf config.Options) *Control {
	return &Control{
		BaseURL:     conf.BaseURL,
		Host:        conf.Host,
		Conf:        conf,
		AuthStorage: authDB,
		Storage:     actorDb,
	}
}

var ctl Control
var logger = logrus.New()

func Before(c *cli.Context) error {
	logger.Level = logrus.ErrorLevel
	ct, err := setup(c, logger)
	if err == nil {
		ctl = *ct
	}

	return err
}

func setup(c *cli.Context, l logrus.FieldLogger) (*Control, error) {
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
	conf, err := config.LoadFromEnv(environ)
	if err != nil {
		l.Errorf("Unable to load config files for environment %s: %s", environ, err)
	}

	host := conf.Host
	var aDb osin.Storage
	var db storage.Repository
	if typ == config.BoltDB {
		path := config.GetDBPath(dir, fmt.Sprintf("%s-oauth", host), environ)
		aDb = auth.NewBoltDBStore(auth.BoltConfig{
			Path:       path,
			BucketName: host,
			LogFn:      func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Infof(s, p...) },
			ErrFn:      func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Errorf(s, p...) },
		})
		db = boltdb.New(boltdb.Config{
			Path:  config.GetDBPath(dir, host, environ),
			LogFn: app.InfoLogFn(l),
			ErrFn: app.ErrLogFn(l),
		}, conf.BaseURL)
		return New(aDb, db, conf), nil
	}
	if typ == config.Badger {
		aDb = auth.NewBoltDBStore(auth.BoltConfig{
			Path:       config.GetDBPath(dir, fmt.Sprintf("%s-oauth", host), environ),
			BucketName: host,
			LogFn: app.InfoLogFn(l),
			ErrFn: app.ErrLogFn(l),
		})
		db = badger.New(badger.Config{
			Path:  dir,
			LogFn: app.InfoLogFn(l),
			ErrFn: app.ErrLogFn(l),
		}, conf.BaseURL)
		return New(aDb, db, conf), nil
	}
	if typ == config.Postgres {
		host := c.String("host")
		if host == "" {
			host = "localhost"
		}
		port := c.Int64("port")
		if port == 0 {
			port = 5432
		}
		user := c.String("user")
		if user == "" {
			user = "fedbox"
		}
		pw, err := loadPwFromStdin(true, "%s@%s's", user, host)
		if err != nil {
			return nil, err
		}
		fedboxDBName := "fedbox"
		oauthDBName := "oauth"
		aDb = auth.NewPgDBStore(auth.PgConfig{
			Enabled: true,
			Host:    host,
			Port:    port,
			User:    user,
			Pw:      string(pw),
			Name:    fedboxDBName,
			LogFn:   app.InfoLogFn(l),
			ErrFn:   app.ErrLogFn(l),
		})
		db, err = pgx.New(config.BackendConfig{
			Enabled: true,
			Host:    host,
			Port:    port,
			User:    user,
			Pw:      string(pw),
			Name:    oauthDBName,
		}, conf.BaseURL, l)
		if err != nil {
			Errf("Error: %s\n", err)
			//return err
		}
	}
	return nil, nil
}

func loadPwFromStdin(confirm bool, s string, params ...interface{}) ([]byte, error) {
	fmt.Printf(s+" pw: ", params...)
	pw1, _ := terminal.ReadPassword(0)
	fmt.Println()
	if confirm {
		fmt.Printf("pw again: ")
		pw2, _ := terminal.ReadPassword(0)
		fmt.Println()
		if !bytes.Equal(pw1, pw2) {
			return nil, errors.Errorf("Passwords do not match")
		}
	}
	return pw1, nil
}

func Errf(s string, par ...interface{}) {
	fmt.Fprintf(os.Stderr, s, par...)
}
