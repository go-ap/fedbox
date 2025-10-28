//go:build storage_sqlite

package storage

import (
	"git.sr.ht/~mariusor/lw"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/internal/config"
	sqlite "github.com/go-ap/storage-sqlite"
)

func Init(c config.Options, l lw.Logger) (FullStorage, error) {
	c.Storage = config.DefaultStorage
	path, err := c.BaseStoragePath()
	if err != nil {
		return nil, err
	}
	l = l.WithContext(lw.Ctx{"path": path})
	l.Debugf("Using sqlite storage")
	db, err := sqlite.New(sqlite.Config{
		Path:        path,
		CacheEnable: c.StorageCache,
		LogFn:       l.Debugf,
		ErrFn:       l.Warnf,
	})

	if err != nil {
		return nil, errors.Annotatef(err, "unable to connect to sqlite storage")
	}
	return db, nil
}

func conf(opt config.Options) sqlite.Config {
	opt.Storage = config.DefaultStorage
	path, _ := opt.BaseStoragePath()
	return sqlite.Config{
		Path:        path,
		CacheEnable: opt.StorageCache,
	}
}

func BootstrapFn(opt config.Options) error {
	return sqlite.Bootstrap(conf(opt))
}

func CleanFn(opt config.Options) error {
	return sqlite.Clean(conf(opt))
}
