//go:build storage_all || (!storage_pgx && !storage_boltdb && !storage_fs && !storage_badger && !storage_sqlite)
// +build storage_all !storage_pgx,!storage_boltdb,!storage_fs,!storage_badger,!storage_sqlite

package cmd

import (
	"fmt"
	"os"

	authsqlite "github.com/go-ap/auth/sqlite"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/storage/badger"
	"github.com/go-ap/fedbox/storage/boltdb"
	"github.com/go-ap/fedbox/storage/fs"
	"github.com/go-ap/fedbox/storage/pgx"
	"github.com/go-ap/fedbox/storage/sqlite"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	bootstrapFn = func(conf config.Options) error {
		if conf.Storage == config.StoragePostgres {
			var pgRoot string
			// ask for root pw
			fmt.Printf("%s password: ", pgRoot)
			pgPw, _ := terminal.ReadPassword(0)
			fmt.Println()
			return pgx.Bootstrap(conf, pgRoot, pgPw)
		}
		if conf.Storage == config.StorageBoltDB {
			return boltdb.Bootstrap(conf)
		}
		if conf.Storage == config.StorageBadger {
			return badger.Bootstrap(conf)
		}
		if conf.Storage == config.StorageFS {
			return fs.Bootstrap(conf)
		}
		if conf.Storage == config.StorageSqlite {
			if err := authsqlite.Bootstrap(authsqlite.Config{Path: conf.BaseStoragePath()}, nil); err != nil {
				return err
			}
			return sqlite.Bootstrap(conf)
		}
		return errors.NotImplementedf("Invalid storage type %s", conf.Storage)
	}
	cleanFn = func(conf config.Options) error {
		if conf.Storage == config.StorageBoltDB {
			return boltdb.Clean(conf)
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
			return fs.Clean(conf)
		}
		if conf.Storage == config.StorageSqlite {
			return sqlite.Clean(conf)
		}
		return errors.NotImplementedf("Invalid storage type %s", conf.Storage)
	}
)
