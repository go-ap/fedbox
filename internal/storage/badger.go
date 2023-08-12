//go:build storage_badger

package storage

import (
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/storage-badger"
)

func conf(opt config.Options) badger.Config {
	return badger.Config{Path: opt.BaseStoragePath(), CacheEnable: opt.StorageCache}
}

func BootstrapFn(opt config.Options) error {
	return badger.Bootstrap(conf(opt))
}

func CleanFn(opt config.Options) error {
	return badger.Clean(conf(opt))
}
