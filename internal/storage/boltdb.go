//go:build storage_boltdb

package storage

import (
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/storage-boltdb"
)

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
