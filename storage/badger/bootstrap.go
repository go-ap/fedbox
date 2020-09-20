// +build storage_badger storage_all

package badger

import (
	"fmt"
	"github.com/dgraph-io/badger/v2"
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/jsonld"
	"os"
	"strings"
)

func Path (c config.Options) (string, error){
	p := fmt.Sprintf("%s/%s/%s", c.StoragePath, c.Env, c.Host)
	crumbs := strings.Split(p, "/")
	for i := range crumbs {
		current := strings.Join(crumbs[:i], "/")
		if current == "" {
			continue
		}
		if _, err := os.Stat(current); os.IsNotExist(err) {
			if err := os.Mkdir(current, 0700); err != nil {
				return "", err
			}
		}
	}
	return p, nil
}

func Bootstrap(conf config.Options) error {
	path, err := Path(conf)
	if err != nil {
		return err
	}
	db, err := badger.Open(badger.DefaultOptions(path))
	if err != nil {
		return errors.Annotatef(err, "could not open db: %s", err)
	}
	defer db.Close()

	return createService(db, activitypub.Self(activitypub.DefaultServiceIRI(conf.BaseURL)))
}

func createService(b *badger.DB, service pub.Service) error {
	raw, err := jsonld.Marshal(service)
	if err != nil {
		return errors.Annotatef(err, "could not marshal service json")
	}
	return b.Update(func(tx *badger.Txn) error {
		var err error
		path := itemPath(service.GetLink())
		fn := func(k []byte, v []byte) {
			if err = tx.Set(k, v); err != nil {
				err = errors.Annotatef(err, "could not create %s path", k)
			}
		}
		fn(getObjectKey(path), raw)
		fn(getObjectKey(itemPath(service.Inbox.GetLink())), emptyCollection)

		return err
	})
}

func Clean(conf config.Options) error {
	path, err := Path(conf)
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
