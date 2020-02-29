package cmd

import (
	"fmt"
	"github.com/go-ap/auth"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
	"github.com/go-ap/fedbox/storage/boltdb"
	"github.com/go-ap/fedbox/storage/pgx"
	"github.com/go-ap/storage"
	"github.com/openshift/osin"
	"github.com/sirupsen/logrus"
	"gopkg.in/urfave/cli.v2"
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
		path := config.GetBoltDBPath(dir, fmt.Sprintf("%s-oauth", host), environ)
		aDb = auth.NewBoltDBStore(auth.BoltConfig{
			Path:       path,
			BucketName: host,
			LogFn:      func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Infof(s, p...) },
			ErrFn:      func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Errorf(s, p...) },
		})
		db = boltdb.New(boltdb.Config{
			Path:  config.GetBoltDBPath(dir, host, environ),
			LogFn: func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Infof(s, p...) },
			ErrFn: func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Errorf(s, p...) },
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
			LogFn:   func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Infof(s, p...) },
			ErrFn:   func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Errorf(s, p...) },
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
