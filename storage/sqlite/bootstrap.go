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
	p, err := getAbsStoragePath(conf.StoragePath)
	if err != nil {
		return err
	}
	if err := mkDirIfNotExists(p); err != nil {
		return err
	}

	r, err := New(Config{
		StoragePath: conf.StoragePath,
		Env:         string(conf.Env),
		BaseURL:     conf.BaseURL,
	})
	if err != nil {
		return err
	}

	conn := r.conn
	exec := func(qRaw string, par ...interface{}) error {
		qSql := fmt.Sprintf(qRaw, par...)
		_, err = conn.Exec(qSql)
		if err != nil {
			return errors.Annotatef(err, "unable to execute: %q", qSql)
		}
		return nil
	}

	if err != nil {
		return err
	}
	err = exec(createActivitypubObjects)
	if err != nil {
		return err
	}
	err = exec(createActivitypubActivities)
	if err != nil {
		return err
	}
	err = exec(createActivitypubActors)
	if err != nil {
		return err
	}
	err = exec(createActivitypubCollections)
	if err != nil {
		return err
	}
	return r.CreateService(activitypub.Self(activitypub.DefaultServiceIRI(conf.BaseURL)))
}

func (r *repo) Reset() { }

