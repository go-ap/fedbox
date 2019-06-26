package boltdb

import (
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/handlers"
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
		// Service actor
		act := tx.Bucket(rootBucket).Bucket([]byte(bucketActors))
		{
			ib, err := act.CreateBucketIfNotExists([]byte(activitypub.ServiceHash))
			if err != nil {
				return errors.Errorf("could not create item bucket: %s", err)
			}
			j := `{"@context": ["https://www.w3.org/ns/activitystreams"],"id": "%s","type": "Service","name": "self", "inbox": "%s/inbox", "following": "%s/following",  "audience": ["https://www.w3.org/ns/activitystreams#Public"]}`
			a := fmt.Sprintf(j, activitypub.DefaultServiceIRI(baseURL), baseURL, baseURL)
			ib.Put([]byte(handlers.Inbox), nil)
			ib.Put([]byte(handlers.Following), nil)

			err = ib.Put([]byte(objectKey), []byte(a))
			if err != nil {
				return errors.Errorf("could not insert entry: %s", err)
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
