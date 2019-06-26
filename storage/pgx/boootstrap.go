package pgx

import (
	"fmt"
	"github.com/gchaincl/dotsql"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/jackc/pgx"
	"github.com/sirupsen/logrus"
)

var RootDb = "postgres"

func openConn(c pgx.ConnConfig) (*pgx.ConnPool, error) {
	return pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig: c,
		MaxConnections: 3,
	})
}

func Bootstrap(conf config.BackendConfig, rootUser string, rootPw []byte, path string) error {
	log := logrus.New()
	var conn *pgx.ConnPool
	var err error

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
	url := activitypub.DefaultServiceIRI("")
	inbox := fmt.Sprintf("%s/inbox", url)
	following := fmt.Sprintf("%s/follwoing", url)
	err = exec("insert-service-actor", activitypub.ServiceHash, url, url, inbox, following, activitypub.ActivityStreamsPublicNS)
	if err != nil {
		return err
	}
	activities := fmt.Sprintf("%s/%s",url, activitypub.ActivitiesType)
	err = exec("insert-activities-collection", activities)
	if err != nil {
		return err
	}
	objects := fmt.Sprintf("%s/%s",url, activitypub.ObjectsType)
	err = exec("insert-objects-collection", objects)
	if err != nil {
		return err
	}
	actors := fmt.Sprintf("%s/%s",url, activitypub.ActorsType)
	err = exec("insert-actors-collection", actors)
	if err != nil {
		return err
	}
	err = exec("insert-service-actor", url, actors)
	if err != nil {
		return err
	}
	return nil
}

func Clean(c config.BackendConfig) error {
	return nil
}
