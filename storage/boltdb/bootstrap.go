package boltdb

import (
	"github.com/boltdb/bolt"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/handlers"
	"github.com/go-ap/jsonld"
)

func Bootstrap(path string, rootBucket []byte, baseURL string) error {
	var err error
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return errors.Annotatef(err, "could not open db")
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		root, err := tx.CreateBucketIfNotExists(rootBucket)
		if err != nil {
			return errors.Annotatef(err, "could not create root bucket")
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
	if err != nil {
		return errors.Annotatef(err, "could not create buckets")
	}

	err = db.Update(func(tx *bolt.Tx) error {
		{
			root := tx.Bucket(rootBucket)
			service := activitypub.Self(activitypub.DefaultServiceIRI(baseURL))
			raw, _ := jsonld.Marshal(service)

			root.Put([]byte(handlers.Inbox), nil)
			//root.Put([]byte(handlers.Following), nil)
			root.Put([]byte(handlers.Outbox), nil)

			err = root.Put([]byte(objectKey), raw)
			if err != nil {
				return errors.Errorf("could not insert entry: %s", err)
			}

			actors := root.Bucket([]byte(bucketActors))
			if actors == nil {
				return errors.Annotatef(err, "could not open %s bucket", bucketActors)
			}
			if !actors.Writable() {
				return errors.Errorf("Non writeable bucket %s", bucketActors)
			}
		}
		return nil
	})
	if err != nil {
		return errors.Annotatef(err, "Unable to update bolt db")
	}

	return nil
}

func Clean(path string, rootBucket []byte) error {
	var err error
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return errors.Annotatef(err, "could not open db")
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket(rootBucket)
	})
	return err
}
