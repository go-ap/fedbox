//go:build storage_sqlite

package cmd

import (
	sqlite "github.com/go-ap/storage-sqlite"
)

func conf(opt storageConf) sqlite.Config {
	return sqlite.Config{
		Path:        opt.Path,
		CacheEnable: opt.CacheEnable,
	}
}

var (
	bootstrapFn = func(opt storageConf) error {
		return sqlite.Bootstrap(conf(opt), opt.BaseURL)
	}
	cleanFn = func(opt storageConf) error {
		return sqlite.Clean(conf(opt))
	}
)
