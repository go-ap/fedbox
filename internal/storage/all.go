//go:build storage_all || !(storage_boltdb || storage_fs || storage_badger || storage_sqlite)

package storage

import (
	"time"

	"git.sr.ht/~mariusor/lw"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/processing"
	"github.com/go-ap/storage-badger"
	"github.com/go-ap/storage-boltdb"
	"github.com/go-ap/storage-fs"
	"github.com/go-ap/storage-sqlite"
)

func Init(c config.Options, l lw.Logger) (FullStorage, error) {
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
	return nil, errors.NotImplementedf("Invalid storage type %s", c.Storage)
}

func getBadgerStorage(c config.Options, l lw.Logger) (FullStorage, error) {
	path, err := c.BaseStoragePath()
	if err != nil {
		return nil, err
	}
	l = l.WithContext(lw.Ctx{"path": path})
	l.Debugf("Using badger storage")
	conf := badger.Config{
		Path:  path,
		LogFn: l.Debugf,
		ErrFn: l.Warnf,
	}
	db, err := badger.New(conf)
	if err != nil {
		return db, err
	}
	return db, nil
}

func getBoltStorage(c config.Options, l lw.Logger) (FullStorage, error) {
	path, err := c.BaseStoragePath()
	if err != nil {
		return nil, err
	}
	l = l.WithContext(lw.Ctx{"path": path})
	l.Debugf("Using boltdb storage")
	db, err := boltdb.New(boltdb.Config{
		Path:  path,
		LogFn: l.Debugf,
		ErrFn: l.Warnf,
	})
	if err != nil {
		return nil, err
	}
	return db, nil
}

func getFsStorage(c config.Options, l lw.Logger) (FullStorage, error) {
	p, err := c.BaseStoragePath()
	if err != nil {
		return nil, err
	}
	l = l.WithContext(lw.Ctx{"path": p})
	l.Debugf("Using fs storage")
	db, err := fs.New(fs.Config{
		Path:        p,
		CacheEnable: c.StorageCache,
		Logger:      l,
		UseIndex:    c.UseIndex,
	})
	if err != nil {
		return nil, err
	}
	return db, nil
}

func getSqliteStorage(c config.Options, l lw.Logger) (FullStorage, error) {
	path, err := c.BaseStoragePath()
	if err != nil {
		return nil, err
	}
	l = l.WithContext(lw.Ctx{"path": path})
	l.Debugf("Using sqlite storage")
	db, err := sqlite.New(sqlite.Config{
		Path:        path,
		CacheEnable: c.StorageCache,
		LogFn:       l.Debugf,
		ErrFn:       l.Warnf,
	})

	if err != nil {
		return nil, errors.Annotatef(err, "unable to connect to sqlite storage")
	}
	return db, nil
}

func BootstrapFn(opt config.Options) error {
	path, err := opt.BaseStoragePath()
	if err != nil {
		return err
	}
	if opt.Storage == config.StorageBoltDB {
		c := boltdb.Config{Path: path}
		return boltdb.Bootstrap(c)
	}
	if opt.Storage == config.StorageBadger {
		c := badger.Config{Path: path, CacheEnable: opt.StorageCache}
		return badger.Bootstrap(c)
	}
	if opt.Storage == config.StorageFS {
		c := fs.Config{Path: path, CacheEnable: opt.StorageCache, UseIndex: opt.UseIndex}
		return fs.Bootstrap(c)
	}
	if opt.Storage == config.StorageSqlite {
		c := sqlite.Config{Path: path, CacheEnable: opt.StorageCache}
		return sqlite.Bootstrap(c)
	}
	return errors.NotImplementedf("Invalid storage type %s", opt.Storage)
}

func CleanFn(opt config.Options) error {
	path, err := opt.BaseStoragePath()
	if err != nil {
		return err
	}
	if opt.Storage == config.StorageBoltDB {
		c := boltdb.Config{Path: path}
		return boltdb.Clean(c)
	}
	if opt.Storage == config.StorageBadger {
		c := badger.Config{Path: path, CacheEnable: opt.StorageCache}
		return badger.Clean(c)
	}
	if opt.Storage == config.StorageFS {
		conf := fs.Config{Path: path, CacheEnable: opt.StorageCache, UseIndex: opt.UseIndex}
		return fs.Clean(conf)
	}
	if opt.Storage == config.StorageSqlite {
		c := sqlite.Config{Path: path, CacheEnable: opt.StorageCache}
		return sqlite.Clean(c)
	}
	return errors.NotImplementedf("Invalid storage type %s", opt.Storage)
}

func CreateService(opt config.Options, self vocab.Item) (err error) {
	path, err := opt.BaseStoragePath()
	if err != nil {
		return err
	}
	var r processing.WriteStore
	if opt.Storage == config.StorageBoltDB {
		c := boltdb.Config{Path: path}
		r, err = boltdb.New(c)
	}
	if opt.Storage == config.StorageBadger {
		c := badger.Config{Path: path, CacheEnable: opt.StorageCache}
		r, err = badger.New(c)
	}
	if opt.Storage == config.StorageFS {
		c := fs.Config{Path: path, CacheEnable: opt.StorageCache, UseIndex: opt.UseIndex}
		r, err = fs.New(c)
	}
	if opt.Storage == config.StorageSqlite {
		c := sqlite.Config{Path: path, CacheEnable: opt.StorageCache}
		r, err = sqlite.New(c)
	}
	if err != nil {
		return err
	}
	self, err = r.Save(self)
	if err != nil {
		return err
	}
	rr, ok := r.(processing.CollectionStore)
	if !ok {
		return nil
	}
	col := func(iri vocab.IRI) vocab.CollectionInterface {
		return &vocab.OrderedCollection{
			ID:           iri,
			Type:         vocab.OrderedCollectionType,
			Published:    time.Now().UTC(),
			AttributedTo: self,
			CC:           vocab.ItemCollection{vocab.PublicNS},
		}
	}
	return vocab.OnActor(self, func(service *vocab.Actor) error {
		var multi error
		for _, stream := range service.Streams {
			if _, err := rr.Create(col(stream.GetID())); err != nil {
				multi = errors.Join(multi, err)
			}
		}
		return multi
	})
}
