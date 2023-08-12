//go:build storage_boltdb

package storage

import (
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/storage-boltdb"
)

func conf(opt config.Options) boltdb.Config {
	return boltdb.Config{Path: opt.BaseStoragePath()}
}

func BootstrapFn(opt config.Options) error {
	return boltdb.Bootstrap(conf(opt))
}

func CleanFn(opt config.Options) error {
	return boltdb.Clean(conf(opt))
}
