// +build storage_sqlite storage_all !sqlite_fs,!storage_boltdb,!storage_badger,!storage_pgx

package sqlite

import (
	"fmt"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/config"
	"os"
	"path"
)

func Clean(conf config.Options) error {
	return os.RemoveAll(path.Join(conf.StoragePath, string(conf.Env)))
}

func Bootstrap(conf config.Options) error {
	Clean(conf)
	p, err := getFullPath(Config{
		StoragePath: conf.StoragePath,
		Env:         string(conf.Env),
		BaseURL:     conf.BaseURL,
	})
	if err != nil {
		return err
	}

	r := repo{
		baseURL: conf.BaseURL,
		path:    p,
		logFn:   defaultLogFn,
		errFn:   defaultLogFn,
	}
	exec := func(qRaw string, par ...interface{}) error {
		if err := r.Open(); err != nil {
			return err
		}
		defer r.Close()
		qSql := fmt.Sprintf(qRaw, par...)
		_, err = r.conn.Exec(qSql)
		if err != nil {
			return errors.Annotatef(err, "unable to execute: %q", qSql)
		}
		return nil
	}

	if err != nil {
		return err
	}
	err = exec(createObjects)
	if err != nil {
		return err
	}
	err = exec(createActivities)
	if err != nil {
		return err
	}
	err = exec(createActors)
	if err != nil {
		return err
	}
	err = exec(createCollections)
	if err != nil {
		return err
	}
	return r.CreateService(activitypub.Self(activitypub.DefaultServiceIRI(conf.BaseURL)))
}

func (r *repo) Reset() { }

