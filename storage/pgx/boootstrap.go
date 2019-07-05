package pgx

import (
	"fmt"
	"github.com/gchaincl/dotsql"
	"github.com/go-ap/activitystreams"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/jsonld"
	"github.com/jackc/pgx"
	"github.com/sirupsen/logrus"
	"path"
)

var RootDb = "postgres"

func openConn(c pgx.ConnConfig) (*pgx.Conn, error) {
	return pgx.Connect(c)
}

func Bootstrap(opt config.Options, rootUser string, rootPw []byte, file string) error {
	log := logrus.New()
	var conn *pgx.Conn
	var err error

	conf := opt.DB
	if conf.User == "" {
		return errors.Newf("empty user")
	}
	if conf.Name == "" {
		return errors.Newf("empty name")
	}
	if conf.Host == "" {
		return errors.Newf("empty host")
	}
	if opt.BaseURL == "" {
		return errors.Newf("empty base URL")
	}

	conn, err = openConn(pgx.ConnConfig{
		Host:     conf.Host,
		Port:     uint16(conf.Port),
		Database: RootDb,
		User:     rootUser,
		Password: string(rootPw),
		Logger:   DBLogger(log),
	})
	if err != nil {
		return err
	}
	dot, err := dotsql.LoadFromFile(file)
	if err != nil {
		return errors.Annotatef(err, "could not open bootstrap file %s", file)
	}

	exec := func(lbl string, par ...interface{}) error {
		qRaw, err := dot.Raw(lbl)
		if err != nil {
			return errors.Annotatef(err, "unable to load query: %s", lbl)
		}
		qSql := fmt.Sprintf(qRaw, par...)
		_, err = conn.Exec(qSql)
		if err != nil {
			return errors.Annotatef(err, "unable to execute: %s", lbl)
		}
		return nil
	}

	// Root queries
	err = exec("create-role-with-pass", conf.User, conf.Pw)
	if err != nil {
		return err
	}
	err = exec("create-db-for-role", conf.Name, conf.User)
	if err != nil {
		return err
	}
	conn.Close()

	// non-root queries
	conn, err = openConn(pgx.ConnConfig{
		Host:     conf.Host,
		Port:     uint16(conf.Port),
		Database: conf.Name,
		User:     conf.User,
		Password: conf.Pw,
		Logger:   DBLogger(log),
	})
	if err != nil {
		return err
	}
	defer conn.Close()
	err = exec("create-activitypub-types-enum")
	if err != nil {
		return err
	}
	err = exec("create-activitypub-objects")
	if err != nil {
		return err
	}
	err = exec("create-activitypub-activities")
	if err != nil {
		return err
	}
	err = exec("create-activitypub-actors")
	if err != nil {
		return err
	}
	err = exec("create-activitypub-collections")
	if err != nil {
		return err
	}
	baseURL := activitypub.DefaultServiceIRI(opt.BaseURL)
	service := activitypub.Self(baseURL)

	u, _ := service.GetLink().URL()
	raw, _ := jsonld.Marshal(service)
	err = exec("insert-actor", path.Base(u.Path), service.GetType(), service.GetLink(), raw)
	if err != nil {
		return err
	}
	activities := fmt.Sprintf("%s%s", baseURL, activitypub.ActivitiesType)
	err = exec("insert-collection", activities, activitystreams.OrderedCollectionType)
	if err != nil {
		return err
	}
	objects := fmt.Sprintf("%s%s", baseURL, activitypub.ObjectsType)
	err = exec("insert-collection", objects, activitystreams.OrderedCollectionType)
	if err != nil {
		return err
	}
	actors := fmt.Sprintf("%s%s", baseURL, activitypub.ActorsType)
	err = exec("insert-collection", actors, activitystreams.OrderedCollectionType)
	if err != nil {
		return err
	}
	//err = exec("add-to-collection", service.GetLink(), actors)
	//if err != nil {
	//	return err
	//}
	err = exec("insert-collection", service.Inbox.GetLink(), activitystreams.OrderedCollectionType)
	if err != nil {
		return err
	}
	//err = exec("insert-collection", service.Following.GetLink(), activitystreams.OrderedCollectionType)
	//if err != nil {
	//	return err
	//}
	err = exec("insert-collection", service.Outbox.GetLink(), activitystreams.OrderedCollectionType)
	if err != nil {
		return err
	}
	return nil
}

func Clean(opt config.Options, rootUser string, rootPw []byte, path string) error {
	log := logrus.New()
	var conn *pgx.Conn
	var err error

	conf := opt.DB
	if conf.User == "" {
		return errors.Newf("empty user")
	}
	if conf.Name == "" {
		return errors.Newf("empty name")
	}
	if conf.Host == "" {
		return errors.Newf("empty host")
	}

	conn, err = openConn(pgx.ConnConfig{
		Host:                 conf.Host,
		Port:                 uint16(conf.Port),
		Database:             RootDb,
		User:                 rootUser,
		Password:             string(rootPw),
		Logger:               DBLogger(log),
		LogLevel:             pgx.LogLevelWarn,
	})
	if err != nil {
		return err
	}
	defer conn.Close()
	dot, err := dotsql.LoadFromFile(path)
	if err != nil {
		return errors.Annotatef(err, "could not open bootstrap file %s", path)
	}

	exec := func(lbl string, par ...interface{}) error {
		qRaw, err := dot.Raw(lbl)
		if err != nil {
			return errors.Annotatef(err, "unable to load query: %s", lbl)
		}
		qSql := fmt.Sprintf(qRaw, par...)
		_, err = conn.Exec(qSql)
		if err != nil {
			return errors.Annotatef(err, "unable to execute: %s", lbl)
		}
		return nil
	}
	err = exec("drop-database", conf.Name)
	if err != nil {
		return err
	}
	err = exec("drop-role", conf.User)
	if err != nil {
		return err
	}
	return nil
}
