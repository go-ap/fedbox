//go:build storage_all || (!storage_boltdb && !storage_fs && !storage_badger && !storage_sqlite)

package fedbox

import (
	"git.sr.ht/~mariusor/lw"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/processing"
	"github.com/go-ap/storage-badger"
	"github.com/go-ap/storage-boltdb"
	"github.com/go-ap/storage-fs"
	"github.com/go-ap/storage-sqlite"
	"github.com/openshift/osin"
)

func getBadgerStorage(c config.Options, l lw.Logger) (processing.Store, osin.Storage, error) {
	path := c.BaseStoragePath()
	l = l.WithContext(lw.Ctx{"path": path})
	l.Debugf("Initializing badger storage")
	conf := badger.Config{
		Path:  path,
		LogFn: l.Debugf,
		ErrFn: l.Warnf,
	}
	db, err := badger.New(conf)
	if err != nil {
		return db, nil, err
	}
	return db, db, nil
}

func getBoltStorage(c config.Options, l lw.Logger) (processing.Store, osin.Storage, error) {
	path := c.BaseStoragePath()
	l = l.WithContext(lw.Ctx{"path": path})
	l.Debugf("Initializing boltdb storage")
	db, err := boltdb.New(boltdb.Config{
		Path:  path,
		LogFn: l.Debugf,
		ErrFn: l.Warnf,
	})
	if err != nil {
		return nil, nil, err
	}
	return db, db, nil
}

func getFsStorage(c config.Options, l lw.Logger) (processing.Store, osin.Storage, error) {
	p := c.BaseStoragePath()
	l = l.WithContext(lw.Ctx{"path": p})
	l.Debugf("Initializing fs storage")
	db, err := fs.New(fs.Config{
		Path:        p,
		CacheEnable: c.StorageCache,
		LogFn:       l.Debugf,
		ErrFn:       l.Warnf,
	})
	if err != nil {
		return nil, nil, err
	}
	return db, db, nil
}

func getSqliteStorage(c config.Options, l lw.Logger) (processing.Store, osin.Storage, error) {
	path := c.BaseStoragePath()
	l = l.WithContext(lw.Ctx{"path": path})
	l.Debugf("Initializing sqlite storage")
	db, err := sqlite.New(sqlite.Config{
		Path:        path,
		CacheEnable: c.StorageCache,
		LogFn:       l.Debugf,
		ErrFn:       l.Warnf,
	})

	if err != nil {
		return nil, nil, errors.Annotatef(err, "unable to connect to sqlite storage")
	}
	return db, db, nil
}

func Storage(c config.Options, l lw.Logger) (processing.Store, osin.Storage, error) {
	switch c.Storage {
	case config.StorageBoltDB:
		return getBoltStorage(c, l)
	case config.StorageBadger:
		return getBadgerStorage(c, l)
	case config.StorageSqlite:
		return getSqliteStorage(c, l)
	case config.StorageFS:
		return getFsStorage(c, l)
	}
	return nil, nil, errors.NotImplementedf("Invalid storage type %s", c.Storage)
}
