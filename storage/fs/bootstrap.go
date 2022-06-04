//go:build storage_fs || storage_all || (!storage_boltdb && !storage_badger && !storage_pgx && !storage_sqlite)

package fs

import (
	"os"
	"path"

	vocab "github.com/go-ap/activitypub"
	ap "github.com/go-ap/fedbox/activitypub"
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
	self := ap.Self(ap.DefaultServiceIRI(conf.BaseURL))
	actors := &vocab.OrderedCollection{ID: ap.ActorsType.IRI(self)}
	if _, err = r.Create(actors); err != nil {
		return err
	}
	activities := &vocab.OrderedCollection{ID: ap.ActivitiesType.IRI(self)}
	if _, err = r.Create(activities); err != nil {
		return err
	}
	objects := &vocab.OrderedCollection{ID: ap.ObjectsType.IRI(self)}
	if _, err = r.Create(objects); err != nil {
		return err
	}
	return nil
}

func (r *repo) Reset() {
	r.cache = cache.New(true)
}
