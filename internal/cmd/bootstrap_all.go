//go:build storage_all || (!storage_boltdb && !storage_fs && !storage_badger && !storage_sqlite)

package cmd

import (
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/storage-badger"
	"github.com/go-ap/storage-boltdb"
	"github.com/go-ap/storage-fs"
	"github.com/go-ap/storage-sqlite"
)

var (
	bootstrapFn = func(conf storageConf, service vocab.Item) error {
		if conf.Storage == config.StorageBoltDB {
			c := boltdb.Config{Path: conf.Path}
			return boltdb.Bootstrap(c, conf.BaseURL)
		}
		if conf.Storage == config.StorageBadger {
			c := badger.Config{Path: conf.Path, CacheEnable: conf.CacheEnable}
			return badger.Bootstrap(c, conf.BaseURL)
		}
		if conf.Storage == config.StorageFS {
			c := fs.Config{Path: conf.Path, CacheEnable: conf.CacheEnable}
			return fs.Bootstrap(c, service)
		}
		if conf.Storage == config.StorageSqlite {
			c := sqlite.Config{Path: conf.Path, CacheEnable: conf.CacheEnable}
			return sqlite.Bootstrap(c, conf.BaseURL)

		}
		return errors.NotImplementedf("Invalid storage type %s", conf.Storage)
	}
	cleanFn = func(conf storageConf) error {
		if conf.Storage == config.StorageBoltDB {
			c := boltdb.Config{Path: conf.Path}
			return boltdb.Clean(c)
		}
		if conf.Storage == config.StorageBadger {
			c := badger.Config{Path: conf.Path, CacheEnable: conf.CacheEnable}
			return badger.Clean(c)
		}
		if conf.Storage == config.StorageFS {
			conf := fs.Config{Path: conf.Path, CacheEnable: conf.CacheEnable}
			return fs.Clean(conf)
		}
		if conf.Storage == config.StorageSqlite {
			conf := sqlite.Config{Path: conf.Path, CacheEnable: conf.CacheEnable}
			return sqlite.Clean(conf)
		}
		return errors.NotImplementedf("Invalid storage type %s", conf.Storage)
	}
)
