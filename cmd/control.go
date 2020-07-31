package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/app"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
	"github.com/go-ap/fedbox/storage/badger"
	"github.com/go-ap/fedbox/storage/boltdb"
	"github.com/go-ap/fedbox/storage/fs"
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
	AuthStorage osin.Storage
	Storage     storage.Repository
}

func New(authDB osin.Storage, actorDb storage.Repository, conf config.Options) *Control {
	return &Control{
		Conf:        conf,
		AuthStorage: authDB,
		Storage:     actorDb,
	}
}

var ctl Control
var logger = logrus.New()

func Before(c *cli.Context) error {
	logger.Level = logrus.WarnLevel
	ct, err := setup(c, logger)
	if err != nil {
		// Ensure we don't print the default help message, which is not useful here
		c.App.CustomAppHelpTemplate = "Failed"
		logger.Errorf("Error: %s", err)
		return err
	}
	ctl = *ct
	// the level enums have same values
	logger.Level = logrus.Level(ct.Conf.LogLevel)

	return nil
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
	conf, err := config.LoadFromEnv(environ)
	if err != nil {
		l.Errorf("Unable to load config files for environment %s: %s", environ, err)
	}
	if dir == "." && conf.StoragePath != os.TempDir() {
		dir = conf.StoragePath
	}
	typ := c.String("type")
	if typ != "" {
		conf.Storage = config.StorageType(typ)
	}
	host := conf.Host
	var aDb osin.Storage
	var db storage.Repository
	switch conf.Storage {
	case config.StorageBoltDB:
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
	case config.StorageFS:
		path := config.GetDBPath(dir, fmt.Sprintf("%s-oauth", host), environ)
		aDb = auth.NewBoltDBStore(auth.BoltConfig{
			Path:       path,
			BucketName: host,
			LogFn:      func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Infof(s, p...) },
			ErrFn:      func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Errorf(s, p...) },
		})
		l.Printf("fs paths: %s %s\n", path, aDb)
		db, err = fs.New(conf)
		return New(aDb, db, conf), err
	case config.StorageBadger:
		aDb = auth.NewBoltDBStore(auth.BoltConfig{
			Path:       config.GetDBPath(dir, fmt.Sprintf("%s-oauth", host), environ),
			BucketName: host,
			LogFn:      app.InfoLogFn(l),
			ErrFn:      app.ErrLogFn(l),
		})
		storagePath, err := badger.Path(conf)
		if err != nil {
			return nil, err
		}
		db = badger.New(badger.Config{
			Path:  storagePath,
			LogFn: app.InfoLogFn(l),
			ErrFn: app.ErrLogFn(l),
		}, conf.BaseURL)
		return New(aDb, db, conf), nil
	case config.StoragePostgres:
		host := c.String("host")
		if host == "" {
			host = "localhost"
		}
		port := c.Int64("port")
		if port == 0 {
			host = dir
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
			return nil, err
		}
	}
	return nil, errors.Newf("invalid storage type %s", typ)
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

func loadFromStdin(s string, params ...interface{}) ([]byte, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf(s+": ", params...)
	input, _ := reader.ReadBytes('\n')
	fmt.Println()
	return input[:len(input)-1], nil
}

func Errf(s string, par ...interface{}) {
	fmt.Fprintf(os.Stderr, s+"\n", par...)
}
