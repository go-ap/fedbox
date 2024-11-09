//go:build storage_all || (!storage_boltdb && !storage_fs && !storage_badger && !storage_sqlite)

package storage

import (
	"errors"
	"time"

	vocab "github.com/go-ap/activitypub"
	http "github.com/go-ap/errors"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/processing"
	"github.com/go-ap/storage-badger"
	"github.com/go-ap/storage-boltdb"
	"github.com/go-ap/storage-fs"
	"github.com/go-ap/storage-sqlite"
)

func BootstrapFn(opt config.Options) error {
	if opt.Storage == config.StorageBoltDB {
		c := boltdb.Config{Path: opt.BaseStoragePath()}
		return boltdb.Bootstrap(c)
	}
	if opt.Storage == config.StorageBadger {
		c := badger.Config{Path: opt.BaseStoragePath(), CacheEnable: opt.StorageCache}
		return badger.Bootstrap(c)
	}
	if opt.Storage == config.StorageFS {
		c := fs.Config{Path: opt.BaseStoragePath(), CacheEnable: opt.StorageCache, UseIndex: opt.UseIndex}
		return fs.Bootstrap(c)
	}
	if opt.Storage == config.StorageSqlite {
		c := sqlite.Config{Path: opt.BaseStoragePath(), CacheEnable: opt.StorageCache}
		return sqlite.Bootstrap(c)
	}
	return http.NotImplementedf("Invalid storage type %s", opt.Storage)
}

func CleanFn(opt config.Options) error {
	if opt.Storage == config.StorageBoltDB {
		c := boltdb.Config{Path: opt.BaseStoragePath()}
		return boltdb.Clean(c)
	}
	if opt.Storage == config.StorageBadger {
		c := badger.Config{Path: opt.BaseStoragePath(), CacheEnable: opt.StorageCache}
		return badger.Clean(c)
	}
	if opt.Storage == config.StorageFS {
		conf := fs.Config{Path: opt.BaseStoragePath(), CacheEnable: opt.StorageCache, UseIndex: opt.UseIndex}
		return fs.Clean(conf)
	}
	if opt.Storage == config.StorageSqlite {
		c := sqlite.Config{Path: opt.BaseStoragePath(), CacheEnable: opt.StorageCache}
		return sqlite.Clean(c)
	}
	return http.NotImplementedf("Invalid storage type %s", opt.Storage)
}

func CreateService(opt config.Options, self vocab.Item) (err error) {
	var r processing.WriteStore
	if opt.Storage == config.StorageBoltDB {
		c := boltdb.Config{Path: opt.BaseStoragePath()}
		r, err = boltdb.New(c)
	}
	if opt.Storage == config.StorageBadger {
		c := badger.Config{Path: opt.BaseStoragePath(), CacheEnable: opt.StorageCache}
		r, err = badger.New(c)
	}
	if opt.Storage == config.StorageFS {
		c := fs.Config{Path: opt.BaseStoragePath(), CacheEnable: opt.StorageCache, UseIndex: opt.UseIndex}
		r, err = fs.New(c)
	}
	if opt.Storage == config.StorageSqlite {
		c := sqlite.Config{Path: opt.BaseStoragePath(), CacheEnable: opt.StorageCache}
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
