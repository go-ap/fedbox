//go:build storage_sqlite || storage_all || (!storage_fs && !storage_boltdb && !storage_badger && !storage_pgx)

package sqlite

import (
	"fmt"
	"os"

	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/internal/config"
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

func Bootstrap(conf config.Options) (err error) {
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
	if err = r.Open(); err != nil {
		return err
	}
	defer func() {
		err = r.Close()
	}()

	exec := func(qRaw string, par ...interface{}) error {
		qSql := fmt.Sprintf(qRaw, par...)
		if _, err = r.conn.Exec(qSql); err != nil {
			return errors.Annotatef(err, "unable to execute: %q", qSql)
		}
		return nil
	}

	if err = exec(createObjectsQuery); err != nil {
		return err
	}
	if err = exec(createActivitiesQuery); err != nil {
		return err
	}
	if err = exec(createActorsQuery); err != nil {
		return err
	}
	if err = exec(createCollectionsQuery); err != nil {
		return err
	}
	if err = exec(tuneQuery); err != nil {
		return err
	}

	return
}

func (r *repo) Reset() {}
