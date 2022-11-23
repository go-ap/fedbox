//go:build storage_boltdb || storage_all || (!storage_pgx && !storage_fs && !storage_badger && !storage_sqlite)

package boltdb

import (
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/jsonld"
	bolt "go.etcd.io/bbolt"
)

func Bootstrap(conf Config, url string) error {
	r, err := New(conf)
	if err != nil {
		return err
	}
	defer r.Close()

	self := ap.Self(ap.DefaultServiceIRI(url))
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

func createService(b *bolt.DB, service vocab.Service) error {
	raw, err := jsonld.Marshal(service)
	if err != nil {
		return errors.Annotatef(err, "could not marshal service json")
	}
	return b.Update(func(tx *bolt.Tx) error {
		root, err := tx.CreateBucketIfNotExists([]byte(rootBucket))
		if err != nil {
			return errors.Annotatef(err, "could not create root bucket")
		}
		path := itemBucketPath(service.GetLink())
		hostBucket, _, err := descendInBucket(root, path, true)
		if err != nil {
			return errors.Annotatef(err, "could not create %s bucket", path)
		}
		err = hostBucket.Put([]byte(objectKey), raw)
		if err != nil {
			return errors.Annotatef(err, "could not save %s[%s]", service.Name, service.Type)
		}
		_, err = root.CreateBucketIfNotExists([]byte(bucketActivities))
		if err != nil {
			return errors.Annotatef(err, "could not create %s bucket", bucketActivities)
		}
		_, err = root.CreateBucketIfNotExists([]byte(bucketActors))
		if err != nil {
			return errors.Annotatef(err, "could not create %s bucket", bucketActors)
		}
		_, err = root.CreateBucketIfNotExists([]byte(bucketObjects))
		if err != nil {
			return errors.Annotatef(err, "could not create %s bucket", bucketObjects)
		}
		return nil
	})
}

func Clean(conf Config) error {
	path, err := Path(conf)
	if err != nil {
		return err
	}
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return errors.Annotatef(err, "could not open db")
	}
	defer db.Close()

	return db.Update(func(tx *bolt.Tx) error {
		tx.DeleteBucket([]byte(rootBucket))
		return nil
	})
}
