package boltdb

import (
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/handlers"
	"github.com/go-ap/jsonld"
	bolt "go.etcd.io/bbolt"
)

func Bootstrap(path string, baseURL string) error {
	var err error
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return errors.Annotatef(err, "could not open db")
	}
	defer db.Close()

	return createService(db, activitypub.Self(activitypub.DefaultServiceIRI(baseURL)))
}

func createService(b *bolt.DB, service pub.Service) error {
	raw, err := jsonld.Marshal(service)
	if err != nil {
		return errors.Annotatef(err, "could not marshal service json")
	}
	err = b.Update(func(tx *bolt.Tx) error {
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
		_, err = hostBucket.CreateBucketIfNotExists([]byte(bucketActivities))
		if err != nil {
			return errors.Annotatef(err, "could not create %s bucket", bucketActivities)
		}
		_, err = hostBucket.CreateBucketIfNotExists([]byte(bucketActors))
		if err != nil {
			return errors.Annotatef(err, "could not create %s bucket", bucketActors)
		}
		_, err = hostBucket.CreateBucketIfNotExists([]byte(bucketObjects))
		if err != nil {
			return errors.Annotatef(err, "could not create %s bucket", bucketObjects)
		}
		return nil
	})
	if err != nil {
		return errors.Annotatef(err, "could not create buckets")
	}

	return nil
}

func Clean(path string) error {
	var err error
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return errors.Annotatef(err, "could not open db")
	}
	defer db.Close()

	return db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket([]byte(rootBucket))
	})
}

// FIXME(marius): I feel like this hasn't been used anywhere and as such might not work
func AddTestMockActor(path string, actor pub.Actor) error {
	var err error
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return errors.Annotatef(err, "could not open db")
	}
	defer db.Close()

	itPath := itemBucketPath(actor.GetLink())

	err = db.Update(func(tx *bolt.Tx) error {
		root := tx.Bucket([]byte(rootBucket))

		raw, _ := jsonld.Marshal(actor)
		actorBucket, _, err := descendInBucket(root, itPath, true)
		actorBucket.Put([]byte(handlers.Inbox), nil)
		actorBucket.Put([]byte(handlers.Outbox), nil)
		actorBucket.Put([]byte(handlers.Following), nil)
		actorBucket.Put([]byte(handlers.Followers), nil)
		actorBucket.Put([]byte(handlers.Liked), nil)
		actorBucket.Put([]byte(handlers.Likes), nil)
		actorBucket.Put([]byte(handlers.Shares), nil)
		if err != nil {
			return errors.Errorf("could not create actor bucket: %s", err)
		}
		err = actorBucket.Put([]byte(objectKey), raw)
		if err != nil {
			return errors.Errorf("could not insert entry: %s", err)
		}

		//actors := hostBucket.Bucket([]byte(bucketActors))
		//if actors == nil {
		//	return errors.Annotatef(err, "could not open %s bucket", bucketActors)
		//}
		//if !actors.Writable() {
		//	return errors.Errorf("Non writeable bucket %s", bucketActors)
		//}
		return nil
	})

	return err
}
