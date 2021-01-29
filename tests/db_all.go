// +build integration
// +build storage_all !storage_pgx,!storage_boltdb,!storage_fs,!storage_badger,!storage_sqlite

package tests

import (
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/storage/badger"
	"github.com/go-ap/fedbox/storage/boltdb"
	"github.com/go-ap/fedbox/storage/fs"
	"github.com/go-ap/fedbox/storage/sqlite"
)

var resetDB = func(opt config.Options) error {
	var err error
	switch opt.Storage {
	case config.StorageBoltDB:
		boltdb.Clean(opt)
		err = boltdb.Bootstrap(opt)
	case config.StorageBadger:
		badger.Clean(opt)
		err = badger.Bootstrap(opt)
	case config.StorageFS:
		fs.Clean(opt)
		err = fs.Bootstrap(opt)
	//case config.StoragePostgres:
	//	pgx.Clean(opt)
	//	err = pgx.Bootstrap(opt)
	case config.StorageSqlite:
		sqlite.Clean(opt)
		err = sqlite.Bootstrap(opt)
	}
	return err
}
