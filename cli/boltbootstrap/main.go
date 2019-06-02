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
				iri := fmt.Sprintf("%s/%s", conf.Host, bucketActivities)
				err := col.Put([]byte(iri), nil)
				if err != nil {
					return fmt.Errorf("could not insert entry: %v", err)
				}
			}
			{
				iri := fmt.Sprintf("%s/%s", conf.Host, bucketActors)
				err := col.Put([]byte(iri), nil)
				if err != nil {
					return fmt.Errorf("could not insert entry: %v", err)
				}
			}
			{
				iri := fmt.Sprintf("%s/%s", conf.Host, bucketObjects)
				err := col.Put([]byte(iri), nil)
				if err != nil {
					return fmt.Errorf("could not insert entry: %v", err)
				}
			}
		}
		// Service actor
		act := tx.Bucket(rootBucket).Bucket([]byte(bucketActors))
		{
			j := `{"@context": ["https://www.w3.org/ns/activitystreams"],"id": "%s/%s/d3ab037c-0f15-4c09-b635-3d6e201c11aa","type": "Service","name": "self","inbox": "%s/inbox", "audience": [
			  "https://www.w3.org/ns/activitystreams#Public"]}`
			a := fmt.Sprintf(j, conf.Host, bucketActors, conf.Host)
			iri := fmt.Sprintf("%s/%s/d3ab037c-0f15-4c09-b635-3d6e201c11aa", conf.Host, bucketActors)
			err := act.Put([]byte(iri), []byte(a))
			if err != nil {
				return fmt.Errorf("could not insert entry: %v", err)
			}
			col := tx.Bucket(rootBucket).Bucket([]byte(bucketCollections))
			actors := []string{iri}
			aBytes, _ := json.Marshal(&actors)
			{
				iri := fmt.Sprintf("%s/%s", conf.Host, bucketActors)
				err := col.Put([]byte(iri), aBytes)
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
