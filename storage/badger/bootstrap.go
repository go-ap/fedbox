package badger

import (
	"bytes"
	"github.com/dgraph-io/badger/v2"
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/jsonld"
)

func Bootstrap(path string, baseURL string) error {
	db, err := badger.Open(badger.DefaultOptions(path))
	if err != nil {
		return errors.Annotatef(err, "could not open db: %s", err)
	}
	defer db.Close()

	return createService(db, activitypub.Self(activitypub.DefaultServiceIRI(baseURL)))
}

func getActivitiesKey(p []byte) []byte {
	return bytes.Join([][]byte{p, []byte(pathActivities)}, []byte{'/'})
}
func getActorsKey(p []byte) []byte {
	return bytes.Join([][]byte{p, []byte(pathActors)}, []byte{'/'})
}
func getObjectsKey(p []byte) []byte {
	return bytes.Join([][]byte{p, []byte(pathObjects)}, []byte{'/'})
}

func createService(b *badger.DB, service pub.Service) error {
	raw, err := jsonld.Marshal(service)
	if err != nil {
		return errors.Annotatef(err, "could not marshal service json")
	}
	err = b.Update(func(tx *badger.Txn) error {
		path := itemPath(service.GetLink())
		err = tx.Set(getActivitiesKey(path), nil)
		if err != nil {
			return errors.Annotatef(err, "could not create %s path", getActivitiesKey(path))
		}
		err = tx.Set(getActorsKey(path), nil)
		if err != nil {
			return errors.Annotatef(err, "could not create %s path", getActorsKey(path))
		}
		err = tx.Set(getObjectsKey(path), nil)
		if err != nil {
			return errors.Annotatef(err, "could not create %s path", getObjectsKey(path))
		}
		err = tx.Set(getObjectKey(path), raw)
		if err != nil {
			return errors.Annotatef(err, "could not save %s[%s] %s", service.Name, service.Type, getObjectKey(path))
		}
		return nil
	})
	if err != nil {
		return errors.Annotatef(err, "could not create paths")
	}

	return nil
}

func Clean(path string) error {
	var err error
	db, err := badger.Open(badger.DefaultOptions(path))
	if err != nil {
		return errors.Annotatef(err, "could not open db")
	}
	defer db.Close()

	return db.Update(func(tx *badger.Txn) error {
		return nil
	})
}
