package tests

import (
	"encoding/json"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/go-ap/errors"
)


const (
	bucketActors      = "actors"
	bucketActivities  = "activities"
	bucketObjects     = "objects"
	bucketCollections = "collections"
)

func bootstrapBolt(path string, rootBucket []byte, baseURL string) error {
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
		_, err = root.CreateBucketIfNotExists([]byte(bucketCollections))
		if err != nil {
			return errors.Annotatef(err, "could not create %s bucket", bucketCollections)
		}
		return nil
	})
	if err != nil {
		return errors.Annotatef(err, "could not create buckets")
	}

	err = db.Update(func(tx *bolt.Tx) error {
		// Create collections
		col := tx.Bucket(rootBucket).Bucket([]byte(bucketCollections))
		{
			{
				iri := fmt.Sprintf("%s/%s", baseURL, bucketActivities)
				err := col.Put([]byte(iri), nil)
				if err != nil {
					return fmt.Errorf("could not insert entry: %v", err)
				}
			}
			{
				iri := fmt.Sprintf("%s/%s", baseURL, bucketActors)
				err := col.Put([]byte(iri), nil)
				if err != nil {
					return fmt.Errorf("could not insert entry: %v", err)
				}
			}
			{
				iri := fmt.Sprintf("%s/%s", baseURL, bucketObjects)
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
			a := fmt.Sprintf(j, baseURL, bucketActors, baseURL)
			iri := fmt.Sprintf("%s/%s/d3ab037c-0f15-4c09-b635-3d6e201c11aa", baseURL, bucketActors)
			err := act.Put([]byte(iri), []byte(a))
			if err != nil {
				return fmt.Errorf("could not insert entry: %v", err)
			}
			col := tx.Bucket(rootBucket).Bucket([]byte(bucketCollections))
			actors := []string{iri}
			aBytes, _ := json.Marshal(&actors)
			{
				iri := fmt.Sprintf("%s/%s", baseURL, bucketActors)
				err := col.Put([]byte(iri), aBytes)
				if err != nil {
					return fmt.Errorf("could not insert entry: %v", err)
				}
			}
		}
		return nil
	})
	if err != nil {
		return errors.Annotatef(err, "Unable to update bolt db")
	}
	
	return nil
}
