package main

import (
	"encoding/json"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/internal/config"
)

func errf(s string, par ...interface{}) {
	fmt.Printf(s, par...)
}

const (
	bucketActors      = "actors"
	bucketActivities  = "activities"
	bucketObjects     = "objects"
	bucketCollections = "collections"
)

func openDb(path string, root []byte) (*bolt.DB, error) {
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, errors.Annotatef(err, "could not open db")
	}
	rootBucket := []byte(root)
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
		_, err = root.CreateBucketIfNotExists([]byte(bucketCollections))
		if err != nil {
			return errors.Annotatef(err, "could not create %s bucket", bucketCollections)
		}
		return nil
	})
	if err != nil {
		return nil, errors.Annotatef(err, "could not set up buckets")
	}
	return db, nil
}

func main() {
	conf, err := config.LoadFromEnv()
	if err != nil {
		errf("Unable to load settings from environment variables: %s", err)
	}

	rootBucket := []byte(conf.Host)
	db, err := openDb(conf.BoltDBPath, rootBucket)
	if err != nil {
		errf("Unable to open bolt db: %s", err)
	}
	err = db.Update(func(tx *bolt.Tx) error {
		// Create collections
		col := tx.Bucket(rootBucket).Bucket([]byte(bucketCollections))
		{
			{
				err := col.Put([]byte("http://fedbox.git/activities"), nil)
				if err != nil {
					return fmt.Errorf("could not insert entry: %v", err)
				}
			}
			{
				err := col.Put([]byte("http://fedbox.git/actors"), nil)
				if err != nil {
					return fmt.Errorf("could not insert entry: %v", err)
				}
			}
			{
				err := col.Put([]byte("http://fedbox.git/objects"), nil)
				if err != nil {
					return fmt.Errorf("could not insert entry: %v", err)
				}
			}
		}
		// Service actor
		act := tx.Bucket(rootBucket).Bucket([]byte(bucketActors))
		{
			a := `{"@context": ["https://www.w3.org/ns/activitystreams"],"id": "http://fedbox.git/actors/d3ab037c-0f15-4c09-b635-3d6e201c11aa","type": "Service","name": "self","inbox": "http://fedbox.git/inbox", "audience": [
			  "https://www.w3.org/ns/activitystreams#Public"]}`
			err := act.Put([]byte("http://fedbox.git/actors/d3ab037c-0f15-4c09-b635-3d6e201c11aa"), []byte(a))
			if err != nil {
				return fmt.Errorf("could not insert entry: %v", err)
			}
			col := tx.Bucket(rootBucket).Bucket([]byte(bucketCollections))
			actors := []string{"http://fedbox.git/actors/d3ab037c-0f15-4c09-b635-3d6e201c11aa"}
			aBytes, _ := json.Marshal(&actors)
			{
				err := col.Put([]byte("http://fedbox.git/actors"), aBytes)
				if err != nil {
					return fmt.Errorf("could not insert entry: %v", err)
				}
			}
		}
		return nil
	})
	if err != nil {
		errf("Unable to update bolt db: %s", err)
	}
}
