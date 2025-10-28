//go:build storage_boltdb

package storage

import (
	"git.sr.ht/~mariusor/lw"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/storage-boltdb"
)

func Init(c config.Options, l lw.Logger) (FullStorage, error) {
	c.Storage = config.DefaultStorage
	path, err := c.BaseStoragePath()
	if err != nil {
		return nil, err
	}
	l = l.WithContext(lw.Ctx{"path": path})
	l.Debugf("Using boltdb storage")
	db, err := boltdb.New(boltdb.Config{
		Path:  path,
		LogFn: l.Debugf,
		ErrFn: l.Warnf,
	})
	if err != nil {
		return nil, err
	}
	return db, nil
}

func conf(opt config.Options) boltdb.Config {
	opt.Storage = config.DefaultStorage
	path, _ := opt.BaseStoragePath()
	return boltdb.Config{Path: path}
}

func BootstrapFn(opt config.Options) error {
	return boltdb.Bootstrap(conf(opt))
}

func CleanFn(opt config.Options) error {
	return boltdb.Clean(conf(opt))
}
