//go:build storage_all || (!storage_pgx && !storage_boltdb && !storage_fs && !storage_badger && !storage_sqlite)

package cmd

import (
	"fmt"
	"os"

	authsqlite "github.com/go-ap/auth/sqlite"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/storage/badger"
	"github.com/go-ap/fedbox/storage/pgx"
	"github.com/go-ap/storage-boltdb"
	fs "github.com/go-ap/storage-fs"
	sqlite "github.com/go-ap/storage-sqlite"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	bootstrapFn = func(conf storageConf) error {
		if conf.Storage == config.StoragePostgres {
			var pgRoot string
			// ask for root pw
			fmt.Printf("%s password: ", pgRoot)
			pgPw, _ := terminal.ReadPassword(0)
			fmt.Println()
			return pgx.Bootstrap(conf, pgRoot, pgPw)
		}
		if conf.Storage == config.StorageBoltDB {
			c := boltdb.Config{Path: conf.Path}
			return boltdb.Bootstrap(c, conf.BaseURL)
		}
		if conf.Storage == config.StorageBadger {
			return badger.Bootstrap(conf)
		}
		if conf.Storage == config.StorageFS {
			c := fs.Config{
				Path:        conf.StoragePath,
				CacheEnable: conf.StorageCache,
			}
			return fs.Bootstrap(c, conf.BaseURL)
		}
		if conf.Storage == config.StorageSqlite {
			if err := authsqlite.Bootstrap(authsqlite.Config{Path: conf.BaseStoragePath()}, nil); err != nil {
				return err
			}
			c := sqlite.Config{
				Path:        conf.StoragePath,
				CacheEnable: conf.StorageCache,
			}
			return sqlite.Bootstrap(c, conf.BaseURL)

		}
		return errors.NotImplementedf("Invalid storage type %s", conf.Storage)
	}
	cleanFn = func(conf storageConf) error {
		if conf.Storage == config.StorageBoltDB {
			c := boltdb.Config{Path: conf.Path}
			return boltdb.Clean(c)
		}
		if conf.Storage == config.StoragePostgres {
			var pgRoot string
			// ask for root pw
			fmt.Printf("%s password: ", pgRoot)
			pgPw, _ := terminal.ReadPassword(0)
			fmt.Println()
			err := pgx.Clean(conf, pgRoot, pgPw)
			if err != nil {
				return errors.Annotatef(err, "Unable to update %s db", conf.Storage)
			}
		}
		if conf.Storage == config.StorageBadger {
			os.RemoveAll(conf.BadgerOAuth2(conf.BaseStoragePath()))
			return badger.Clean(conf)
		}
		if conf.Storage == config.StorageFS {
			conf := fs.Config{
				Path:        conf.StoragePath,
				CacheEnable: conf.StorageCache,
			}
			return fs.Clean(conf)
		}
		if conf.Storage == config.StorageSqlite {
			conf := sqlite.Config{
				Path:        conf.StoragePath,
				CacheEnable: conf.StorageCache,
			}
			return sqlite.Clean(conf)
		}
		return errors.NotImplementedf("Invalid storage type %s", conf.Storage)
	}
)
