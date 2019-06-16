package boltdb

import (
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/go-ap/errors"
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
			hash := "d3ab037c-0f15-4c09-b635-3d6e201c11aa"
			iri := fmt.Sprintf("%s/%s/%s", baseURL, bucketActors, hash)
			j := `{"@context": ["https://www.w3.org/ns/activitystreams"],"id": "%s","type": "Service","name": "self","inbox": "%s/inbox", "audience": ["https://www.w3.org/ns/activitystreams#Public"]}`
			a := fmt.Sprintf(j, iri, baseURL)
			err := act.Put([]byte(hash), []byte(a))
			if err != nil {
				return fmt.Errorf("could not insert entry: %v", err)
			}
		}
		return nil
	})
	if err != nil {
		return errors.Annotatef(err, "Unable to update bolt db")
	}

	return nil
}
