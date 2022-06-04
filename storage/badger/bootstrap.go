//go:build storage_badger || storage_all || (!storage_pgx && !storage_boltdb && !storage_fs && !storage_sqlite)

package badger

import (
	"fmt"
	"os"

	vocab "github.com/go-ap/activitypub"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/jsonld"
)

var encodeFn = jsonld.Marshal
var decodeFn = jsonld.Unmarshal

func Bootstrap(conf config.Options) error {
	r, err := New(Config{
		Path:    conf.BaseStoragePath(),
		BaseURL: conf.BaseURL,
	})
	if err != nil {
		return err
	}
	self := ap.Self(ap.DefaultServiceIRI(conf.BaseURL))
	actors := &vocab.OrderedCollection{ID: ap.ActorsType.IRI(&self)}
	activities := &vocab.OrderedCollection{ID: ap.ActivitiesType.IRI(&self)}
	objects := &vocab.OrderedCollection{ID: ap.ObjectsType.IRI(&self)}
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
		Path:    conf.BaseStoragePath(),
		BaseURL: conf.BaseURL,
	})
	if err != nil {
		return fmt.Errorf("unable to update %s db: %w", conf.Storage, err)
	}

	return os.RemoveAll(path)
}
