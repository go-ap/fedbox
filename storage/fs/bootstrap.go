// +build storage_fs storage_all !storage_boltdb,!storage_badger,!storage_pgx,!storage_sqlite

package fs

import (
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/cache"
	"github.com/go-ap/fedbox/internal/config"
	"os"
	"path"
)

func Clean(conf config.Options) error {
	return os.RemoveAll(path.Join(conf.StoragePath, string(conf.Env)))
}

func Bootstrap(conf config.Options) error {
	r, err := New(Config{
		StoragePath: conf.StoragePath,
		Env:         string(conf.Env),
		BaseURL:     conf.BaseURL,
	})
	if err != nil {
		return err
	}
	err = r.Open()
	if err != nil {
		return err
	}
	defer r.Close()
	self := activitypub.Self(activitypub.DefaultServiceIRI(conf.BaseURL))
	err = r.CreateService(self)
	if err != nil {
		return err
	}
	actors := &pub.OrderedCollection{ ID: activitypub.ActorsType.IRI(&self) }
	activities := &pub.OrderedCollection{ ID: activitypub.ActivitiesType.IRI(&self) }
	objects := &pub.OrderedCollection{ ID: activitypub.ObjectsType.IRI(&self) }
	if _, err = r.Create(actors); err != nil {
		return err
	}
	if _, err = r.Create(activities); err != nil {
		return err
	}
	if _, err = r.Create(objects); err != nil {
		return err
	}
	return nil
}

func (r *repo) Reset() {
	r.cache = cache.New(true)
}

