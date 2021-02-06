// +build storage_badger storage_all !storage_pgx,!storage_boltdb,!storage_fs,!storage_sqlite

package badger

import (
	"fmt"
	"github.com/dgraph-io/badger/v3"
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/cache"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/jsonld"
	"os"
)

var encodeFn = jsonld.Marshal
var decodeFn = jsonld.Unmarshal

func Bootstrap(conf config.Options) error {
	r, err := New(Config{
		Path:    conf.StoragePath,
		Env:     string(conf.Env),
		BaseURL: conf.BaseURL,
	})
	self := activitypub.Self(activitypub.DefaultServiceIRI(conf.BaseURL))
	actors := &pub.OrderedCollection{ID: activitypub.ActorsType.IRI(&self)}
	activities := &pub.OrderedCollection{ID: activitypub.ActivitiesType.IRI(&self)}
	objects := &pub.OrderedCollection{ID: activitypub.ObjectsType.IRI(&self)}
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

func Clean(conf config.Options) error {
	path, err := Path(Config{
		Path:    conf.StoragePath,
		Env:     string(conf.Env),
		BaseURL: conf.BaseURL,
	})
	if err != nil {
		return fmt.Errorf("unable to update %s db: %w", conf.Storage, err)
	}
	db, err := badger.Open(badger.DefaultOptions(path))
	if err != nil {
		return errors.Annotatef(err, "could not open db %s", path)
	}
	db.Close()

	return os.RemoveAll(path)
}

func (r *repo) Reset() {
	r.cache = cache.New(true)
}
