//go:build storage_boltdb || storage_all || (!storage_pgx && !storage_fs && !storage_badger && !storage_sqlite)

package boltdb

import (
	"os"
	"testing"

	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/internal/config"
	bolt "go.etcd.io/bbolt"
)

func TestBootstrap(t *testing.T) {
	dir := os.TempDir()
	bucket := []byte(rootBucket)
	url := "random-string-not-an-URL"
	conf := config.Options{
		StoragePath: dir,
		Host:        "example.com",
		BaseURL:     url,
	}
	path, _ := Path(Config{
		Path:    dir,
		BaseURL: url,
	})
	err := Bootstrap(conf)
	if err != nil {
		t.Errorf("Error received when cleaning valid boltdb %s with valid root bucket %s: %s", path, bucket, err)
	}

	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		t.Errorf("Could not open boltdb path %s: %s", path, err)
	}

	err = db.View(func(tx *bolt.Tx) error {
		root := tx.Bucket(bucket)
		if false {
			// NOTICE(marius): these have been disabled in the bootstrap, because they're dynamically created
			activities := root.Bucket([]byte(bucketActivities))
			if activities == nil {
				t.Errorf("Could not find bucket %s/%s at boltdb path %s", bucket, bucketActivities, path)
				return nil
			}
			actors := root.Bucket([]byte(bucketActors))
			if actors == nil {
				t.Errorf("Could not find bucket %s/%s at boltdb path %s", bucket, bucketActors, path)
				return nil
			}
			objects := root.Bucket([]byte(bucketObjects))
			if objects == nil {
				t.Errorf("Could not find bucket %s/%s at boltdb path %s", bucket, bucketObjects, path)
				return nil
			}
		}
		return nil
	})
	db.Close()
	err = os.Remove(path)
	if err != nil {
		t.Logf("Unable to clean boltdb path %s", path)
	}
}

func TestClean(t *testing.T) {
	dir := os.TempDir()
	url := "random-string-not-an-URL"
	conf := config.Options{
		StoragePath: dir,
		Host:        "example.com",
		BaseURL:     url,
	}
	path, _ := Path(Config{
		Path:    dir,
		BaseURL: url,
	})
	{
		Clean(conf)
	}
	{
		db, err := bolt.Open(path, 0600, nil)
		if err != nil {
			t.Errorf("Unable to create boltdb at path %s", path)
		}
		db.Close()

		err = Clean(conf)
	}

	{
		db, err := bolt.Open(path, 0600, nil)
		if err != nil {
			t.Errorf("Unable to create boltdb at path %s", path)
		}
		err = db.Update(func(tx *bolt.Tx) error {
			_, err := tx.CreateBucketIfNotExists([]byte(rootBucket))
			if err != nil {
				return errors.Annotatef(err, "could not create root bucket")
			}
			return nil
		})
		if err != nil {
			t.Errorf("Unable to create root bucket %s in boltdb %s", rootBucket, path)
		}
		db.Close()

		err = Clean(conf)
		if err != nil {
			t.Errorf("Error received when cleaning valid boltdb %s with valid root bucket %s: %s", path, rootBucket, err)
		}
	}
	err := os.Remove(path)
	if err != nil {
		t.Logf("Unable to clean boltdb path %s", path)
	}
}
