// +build storage_boltdb storage_all !storage_pgx,!storage_fs,!storage_badger,!storage_sqlite

package boltdb

import (
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/handlers"
	"github.com/go-ap/jsonld"
	bolt "go.etcd.io/bbolt"
)

func Bootstrap(conf config.Options) error {
	r, err := New(Config{
		Path:  conf.StoragePath,
		Env:   string(conf.Env),
		BaseURL: conf.BaseURL,
	})
	if err != nil {
		return err
	}
	defer r.Close()

	self := activitypub.Self(activitypub.DefaultServiceIRI(conf.BaseURL))
	actors := &pub.OrderedCollection{ ID: activitypub.ActorsType.IRI(&self) }
	activities := &pub.OrderedCollection{ ID: activitypub.ActivitiesType.IRI(&self) }
	objects := &pub.OrderedCollection{ ID: activitypub.ObjectsType.IRI(&self) }
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

func createService(b *bolt.DB, service pub.Service) error {
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

func Clean(conf config.Options) error {
	path, err := Path(Config{
		Path:    conf.StoragePath,
		Env:     string(conf.Env),
		BaseURL: conf.BaseURL,
	})
	if err != nil {
		return err
	}
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

