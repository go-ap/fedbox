//go:build storage_badger

package storage

import (
	"git.sr.ht/~mariusor/lw"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/storage-badger"
)

func Init(c config.Options, l lw.Logger) (FullStorage, error) {
	c.Storage = config.DefaultStorage
	path, err := c.BaseStoragePath()
	if err != nil {
		return nil, err
	}
	l = l.WithContext(lw.Ctx{"path": path})
	l.Debugf("Using badger storage")
	conf := badger.Config{
		Path:  path,
		LogFn: l.Debugf,
		ErrFn: l.Warnf,
	}
	db, err := badger.New(conf)
	if err != nil {
		return db, err
	}
	return db, nil
}

func conf(opt config.Options) badger.Config {
	opt.Storage = config.DefaultStorage
	path, _ := opt.BaseStoragePath()
	return badger.Config{Path: path, CacheEnable: opt.StorageCache}
}

func BootstrapFn(opt config.Options) error {
	return badger.Bootstrap(conf(opt))
}

func CleanFn(opt config.Options) error {
	return badger.Clean(conf(opt))
}
