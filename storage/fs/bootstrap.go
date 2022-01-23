//go:build storage_fs || storage_all || (!storage_boltdb && !storage_badger && !storage_pgx && !storage_sqlite)
// +build storage_fs storage_all !storage_boltdb,!storage_badger,!storage_pgx,!storage_sqlite

package fs

import (
	"os"
	"path"

	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/cache"
	"github.com/go-ap/fedbox/internal/config"
)

func Clean(conf config.Options) error {
	return os.RemoveAll(conf.BaseStoragePath())
}

func Bootstrap(conf config.Options) error {
	r, err := New(Config{
		StoragePath: path.Dir(conf.BaseStoragePath()),
		BaseURL:     conf.BaseURL,
		EnableCache: conf.StorageCache,
	})
	if err != nil {
		return err
	}
	err = r.Open()
	if err != nil {
		return err
	}
	defer r.Close()
	self, err := r.Save(activitypub.Self(activitypub.DefaultServiceIRI(conf.BaseURL)))
	if err != nil {
		return err
	}
	actors := &pub.OrderedCollection{ID: activitypub.ActorsType.IRI(self)}
	if _, err = r.Create(actors); err != nil {
		return err
	}
	activities := &pub.OrderedCollection{ID: activitypub.ActivitiesType.IRI(self)}
	if _, err = r.Create(activities); err != nil {
		return err
	}
	objects := &pub.OrderedCollection{ID: activitypub.ObjectsType.IRI(self)}
	if _, err = r.Create(objects); err != nil {
		return err
	}
	return nil
}

func (r *repo) Reset() {
	r.cache = cache.New(true)
}
