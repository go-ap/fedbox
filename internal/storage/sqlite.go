//go:build storage_sqlite

package storage

import (
	"github.com/go-ap/fedbox/internal/config"
	sqlite "github.com/go-ap/storage-sqlite"
)

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
