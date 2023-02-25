//go:build storage_sqlite

package cmd

import (
	vocab "github.com/go-ap/activitypub"
	sqlite "github.com/go-ap/storage-sqlite"
)

func conf(opt storageConf) sqlite.Config {
	return sqlite.Config{
		Path:        opt.Path,
		CacheEnable: opt.CacheEnable,
	}
}

var (
	bootstrapFn = func(opt storageConf, service vocab.Item) error {
		return sqlite.Bootstrap(conf(opt), service)
	}
	cleanFn = func(opt storageConf) error {
		return sqlite.Clean(conf(opt))
	}
)
