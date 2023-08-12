//go:build storage_fs

package storage

import (
	"github.com/go-ap/fedbox/internal/config"
	fs "github.com/go-ap/storage-fs"
)

func conf(opt config.Options) fs.Config {
	return fs.Config{
		Path:        opt.BaseStoragePath(),
		CacheEnable: opt.StorageCache,
	}
}

func BootstrapFn(opt config.Options) error {
	return fs.Bootstrap(conf(opt))
}

func CleanFn(opt config.Options) error {
	return fs.Clean(conf(opt))
}
