//go:build storage_sqlite

package storage

import (
	"github.com/go-ap/fedbox/internal/config"
	sqlite "github.com/go-ap/storage-sqlite"
)

func conf(opt config.Options) sqlite.Config {
	return sqlite.Config{
		Path:        opt.BaseStoragePath(),
		CacheEnable: opt.StorageCache,
	}
}

func BootstrapFn(opt config.Options) error {
	return sqlite.Bootstrap(conf(opt))
}

func CleanFn(opt config.Options) error {
	return sqlite.Clean(conf(opt))
}
