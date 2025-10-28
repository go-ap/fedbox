//go:build storage_fs

package storage

import (
	"git.sr.ht/~mariusor/lw"
	"github.com/go-ap/fedbox/internal/config"
	fs "github.com/go-ap/storage-fs"
)

func Init(c config.Options, l lw.Logger) (FullStorage, error) {
	c.Storage = config.DefaultStorage
	p, err := c.BaseStoragePath()
	if err != nil {
		return nil, err
	}
	l = l.WithContext(lw.Ctx{"path": p})
	l.Debugf("Using fs storage")
	db, err := fs.New(fs.Config{
		Path:        p,
		CacheEnable: c.StorageCache,
		Logger:      l,
		UseIndex:    c.UseIndex,
	})
	if err != nil {
		return nil, err
	}
	return db, nil
}

func conf(opt config.Options) fs.Config {
	opt.Storage = config.DefaultStorage
	path, _ := opt.BaseStoragePath()
	return fs.Config{
		Path:        path,
		CacheEnable: opt.StorageCache,
		UseIndex:    opt.UseIndex,
	}
}

func BootstrapFn(opt config.Options) error {
	return fs.Bootstrap(conf(opt))
}

func CleanFn(opt config.Options) error {
	return fs.Clean(conf(opt))
}
