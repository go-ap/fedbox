// +build storage_sqlite storage_all !sqlite_fs,!storage_boltdb,!storage_badger,!storage_pgx

package sqlite

import (
	"fmt"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/internal/config"
	"os"
)

func Clean(conf config.Options) error {
	p, err := getFullPath(Config{
		StoragePath: conf.BaseStoragePath(),
		BaseURL:     conf.BaseURL,
	})
	if err != nil {
		return err
	}
	return os.RemoveAll(p)
}

func Bootstrap(conf config.Options) error {
	Clean(conf)

	p, err := getFullPath(Config{
		StoragePath: conf.BaseStoragePath(),
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
		_, err := r.conn.Exec(qSql)
		if err != nil {
			return errors.Annotatef(err, "unable to execute: %q", qSql)
		}
		return nil
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
	return nil
}

func (r *repo) Reset() { }

