package boltdb

import (
	"bytes"
	"fmt"
	"github.com/boltdb/bolt"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/handlers"
	"github.com/go-ap/jsonld"
	s "github.com/go-ap/storage"
	"github.com/sirupsen/logrus"
	"net/url"
	"strings"
	"time"
)

type repo struct {
	d       *bolt.DB
	baseURL string
	root    []byte
	path    string
	logFn   loggerFn
	errFn   loggerFn
}

type loggerFn func(logrus.Fields, string, ...interface{})

const (
	bucketActors     = "actors"
	bucketActivities = "activities"
	bucketObjects    = "objects"
)

// Config
type Config struct {
	Path       string
	BucketName string
	LogFn      loggerFn
	ErrFn      loggerFn
}

// New returns a new repo repository
func New(c Config, baseURL string) (*repo, error) {
	b := repo{
		root:    []byte(c.BucketName),
		path:    c.Path,
		baseURL: baseURL,
		logFn:   func(logrus.Fields, string, ...interface{}) {},
		errFn:   func(logrus.Fields, string, ...interface{}) {},
	}
	if c.ErrFn != nil {
		b.errFn = c.ErrFn
	}
	if c.LogFn != nil {
		b.logFn = c.LogFn
	}
	return &b, nil
}

func loadFromBucket(db *bolt.DB, root []byte, f s.Filterable) (as.ItemCollection, uint, error) {
	col := make(as.ItemCollection, 0)

	err := db.View(func(tx *bolt.Tx) error {
		rb := tx.Bucket(root)
		if rb == nil {
			return errors.Errorf("Invalid bucket %s", root)
		}
		iri := f.ID()
		url, err := iri.URL()
		if err != nil {
			return errors.Newf("invalid IRI filter element %s when loading collections", iri)
		}
		if string(root) != url.Host {
			return errors.Newf("trying to load from non-local root bucket %s", url.Host)
		}
		// Assume bucket exists and has keys
		b, path, err := descendInBucket(rb, url.Path)
		if err != nil {
			return err
		}
		if b == nil {
			return errors.Errorf("Invalid bucket %s/%s", root, path)
		}

		c := b.Cursor()
		if c == nil {
			return errors.Errorf("Invalid bucket cursor %s/%s", root, path)
		}
		if path == "" {
			path = objectKey
		}
		prefix := []byte(path)
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			if it, err := as.UnmarshalJSON(v); err == nil {
				col = append(col, it)
			}
		}
		return nil
	})

	return col, uint(len(col)), err
}

// Load
func (b *repo) Load(f s.Filterable) (as.ItemCollection, uint, error) {
	var err error
	err = b.Open()
	if err != nil {
		return nil, 0, err
	}
	defer b.Close()
	return nil, 0, errors.NotImplementedf("BoltDB Load not implemented")
}

// LoadActivities
func (b *repo) LoadActivities(f s.Filterable) (as.ItemCollection, uint, error) {
	var err error
	err = b.Open()
	if err != nil {
		return nil, 0, err
	}
	defer b.Close()
	return loadFromBucket(b.d, b.root, f)
}

// LoadObjects
func (b *repo) LoadObjects(f s.Filterable) (as.ItemCollection, uint, error) {
	var err error
	err = b.Open()
	if err != nil {
		return nil, 0, err
	}
	defer b.Close()
	return loadFromBucket(b.d, b.root, f)
}

// LoadActors
func (b *repo) LoadActors(f s.Filterable) (as.ItemCollection, uint, error) {
	var err error
	err = b.Open()
	if err != nil {
		return nil, 0, err
	}
	defer b.Close()
	return loadFromBucket(b.d, b.root, f)
}

func descendInBucket(root *bolt.Bucket, path string) (*bolt.Bucket, string, error) {
	if root == nil {
		return nil, path, errors.Newf("Trying to descend into nil bucket")
	}
	if len(path) == 0 {
		return nil, path, errors.Newf("Trying to descend into nil bucket tree")
	}
	buckets := strings.Split(path, "/")

	lvl := 0
	b := root
	// descend the bucket tree up to the last found bucket
	for _, name := range buckets {
		lvl++
		if len(name) == 0 {
			continue
		}
		if b == nil {
			return root, path, errors.Errorf("trying to load from nil bucket")
		}
		cb := b.Bucket([]byte(name))
		if cb == nil {
			lvl--
			break
		}
		b = cb
	}
	path = strings.Join(buckets[lvl:], "/")

	return b, path, nil
}

// LoadCollection
func (b *repo) LoadCollection(f s.Filterable) (as.CollectionInterface, error) {
	var err error
	err = b.Open()
	if err != nil {
		return nil, err
	}
	defer b.Close()

	var ret as.CollectionInterface
	iri := f.ID()
	url, err := iri.URL()
	if err != nil {
		b.errFn(nil, "invalid IRI filter element %s when loading collections", iri)
	}
	if string(b.root) != url.Host {
		b.errFn(nil, "trying to load from non-local root bucket %s", url.Host)
	}
	col := &as.OrderedCollection{}
	col.ID = as.ObjectID(iri)
	col.Type = as.OrderedCollectionType

	err = b.d.View(func(tx *bolt.Tx) error {
		rb := tx.Bucket(b.root)
		if rb == nil {
			return errors.Newf("invalid root bucket %s", b.root)
		}
		bb, path, err := descendInBucket(rb, url.Path)
		if err != nil {
			b.errFn(nil, "unable to find %s in root bucket", path, b.root)
		}
		if len(path) == 0 {
			cb := bb.Cursor()
			if cb == nil {
				return errors.Errorf("Invalid collection bucket path %s", path)
			}
			for uuid, _ := cb.First(); uuid != nil; uuid, _ = cb.Next() {
				ib := bb.Bucket(uuid)
				raw := ib.Get([]byte(objectKey))
				if raw == nil || len(raw) == 0 {
					return errors.Annotatef(err, "empty raw item")
				}

				it, err := as.UnmarshalJSON(raw)
				if err != nil {
					return errors.Annotatef(err, "unable to unmarshal raw item")
				}
				if err = col.Append(it); err == nil {
					col.TotalItems++
				}
			}
		} else {
			raw := bb.Get([]byte(path))
			if raw == nil || len(raw) == 0 {
				return nil
			}
			return errors.NotImplementedf("TODO: unmarshall collection items in collection: %s", url.Path)
			// This should be a marshalled json array of IRIs,
			// I probably need to use a different marshalling/unmarshalling method than the activitystreams one.
			// Then for each element in the array, we load the corresponding item from the base collection
			// The base collection in this case is one of: actors, activities, objects
			it, err := as.UnmarshalJSON(raw)
			if err != nil {
				return errors.Annotatef(err, "unable to unmarshal raw item")
			}
			if err = col.Append(it); err == nil {
				col.TotalItems++
			}
		}
		return err
	})
	ret = col
	return ret, err
}

const objectKey = "__raw"

func delete(db *bolt.DB, rootBkt []byte, it as.Item) (as.Item, error) {
	f := ap.Filters{
		IRI: it.GetLink(),
	}
	if it.IsObject() {
		f.Type = []as.ActivityVocabularyType{it.GetType()}
	}
	var cnt uint
	var found as.ItemCollection
	found, cnt, _ = loadFromBucket(db, rootBkt, &f)
	if cnt == 0 {
		err := errors.NotFoundf("%s in either actors or objects", it.GetLink())
		return it, err
	}
	old := found.First()

	t := as.Tombstone{
		Parent: as.Parent{
			ID:   as.ObjectID(it.GetLink()),
			Type: as.TombstoneType,
			To: as.ItemCollection{
				ap.Public,
			},
		},
		Deleted:    time.Now().UTC(),
		FormerType: old.GetType(),
	}
	return save(db, rootBkt, t)
}

func getCollectionID(actor as.Item, c handlers.CollectionType) as.ObjectID {
	return as.ObjectID(fmt.Sprintf("%s/%s", actor.GetLink(), c))
}

func createObjectCollection(b *bolt.Bucket, it as.Item, c handlers.CollectionType) (as.CollectionInterface, error) {
	col := as.OrderedCollection{
		Parent: as.Parent{
			ID:   getCollectionID(it, c),
			Type: as.OrderedCollectionType,
		},
	}
	b.Put([]byte(c), nil)
	return &col, nil
}

func getCollectionIRI(actor as.Item, c handlers.CollectionType) as.IRI {
	return as.IRI(fmt.Sprintf("%s/%s", actor.GetLink(), c))
}
func save(db *bolt.DB, rootBkt []byte, it as.Item) (as.Item, error) {
	url, err := it.GetLink().URL()
	if err != nil {
		return it, errors.Annotatef(err, "invalid IRI")
	}
	path := url.Path

	var uuid string
	err = db.Update(func(tx *bolt.Tx) error {
		root := tx.Bucket(rootBkt)
		if root == nil {
			return errors.Errorf("Invalid bucket %s", rootBkt)
		}
		if !root.Writable() {
			return errors.Errorf("Non writeable bucket %s", rootBkt)
		}
		var b *bolt.Bucket
		b, uuid, err = descendInBucket(root, path)
		if err != nil {
			return errors.Newf("Unable to find %s in root bucket", path)
		}
		if !b.Writable() {
			return errors.Errorf("Non writeable bucket %s", path)
		}
		if uuid != "" {
			b, err = b.CreateBucket([]byte(uuid))
			if err != nil {
				return errors.Errorf("could not create item bucket entry: %s", err)
			}
		}
		if as.ActorTypes.Contains(it.GetType()) {
			if p, err := ap.ToPerson(it); err == nil {
				if in, err := createObjectCollection(b, p, handlers.Inbox); err != nil {
					return errors.Errorf("could not create bucket for collection %s", err)
				} else {
					p.Inbox = in.GetLink()
				}
				if out, err := createObjectCollection(b, p, handlers.Outbox); err != nil {
					return errors.Errorf("could not create bucket for collection %s", err)
				} else {
					p.Outbox = out.GetLink()
				}
				if fers, err := createObjectCollection(b, p, handlers.Followers); err != nil {
					return errors.Errorf("could not create bucket for collection %s", err)
				} else {
					p.Followers = fers.GetLink()
				}
				if fing, err := createObjectCollection(b, p, handlers.Following); err != nil {
					return errors.Errorf("could not create bucket for collection %s", err)
				} else {
					p.Following = fing.GetLink()
				}
				if ld, err := createObjectCollection(b, p, handlers.Liked); err != nil {
					return errors.Errorf("could not create bucket for collection %s", err)
				} else {
					p.Liked = ld.GetLink()
				}
				if ls, err := createObjectCollection(b, p, handlers.Likes); err != nil {
					return errors.Errorf("could not create bucket for collection %s", err)
				} else {
					p.Likes = ls.GetLink()
				}
				if sh, err := createObjectCollection(b, p, handlers.Shares); err != nil {
					return errors.Errorf("could not create bucket for collection %s", err)
				} else {
					p.Shares = sh.GetLink()
				}
				it = p
			}
		} else if as.ObjectTypes.Contains(it.GetType()) {
			if o, err := as.ToObject(it); err == nil {
				if repl, err := createObjectCollection(b, o, handlers.Replies); err != nil {
					return errors.Errorf("could not create bucket for collection %s", err)
				} else {
					o.Replies = repl.GetLink()
				}
				it = o
			}
		}
		entryBytes, err := jsonld.Marshal(it)
		if err != nil {
			return errors.Annotatef(err, "could not marshal activity")
		}
		err = b.Put([]byte(objectKey), entryBytes)
		if err != nil {
			return errors.Errorf("could not insert entry: %s", err)
		}

		return nil
	})

	return it, err
}

// SaveActivity
func (b *repo) SaveActivity(it as.Item) (as.Item, error) {
	var err error
	if it, err = b.SaveObject(it); err == nil {
		b.logFn(nil, "Added new activity: %s", it.GetLink())
	}
	return it, err
}

// SaveObject
func (b *repo) SaveObject(it as.Item) (as.Item, error) {
	var err error
	err = b.Open()
	if err != nil {
		return it, err
	}
	defer b.Close()
	var bucket string
	if as.ActivityTypes.Contains(it.GetType()) {
		bucket = bucketActivities
	} else if as.ActorTypes.Contains(it.GetType()) {
		bucket = bucketActors
	} else {
		bucket = bucketObjects
	}
	iri := it.GetLink()
	if len(iri) == 0 {
		pc := as.IRI(fmt.Sprintf("%s/%s", b.baseURL, bucket))
		if _, err = b.GenerateID(it, pc, nil); err != nil {
			return it, err
		}
	}
	if it, err = save(b.d, b.root, it); err == nil {
		b.logFn(nil, "Added new %s: %s", bucket[:len(bucket)-1], it.GetLink())
	}

	// TODO(marius) Move to somewhere else
	if toFw, ok := it.(as.HasRecipients); ok {
		for _, fw := range toFw.Recipients() {
			colIRI := fw.GetLink()
			if b.IsLocalIRI(colIRI) {
				// we shadow the err variable intentionally so it does not propagate upper to the call stack
				if errFw := addToCollection(b.d, b.root, colIRI, it); err != nil {
					b.errFn(logrus.Fields{"IRI": it.GetLink(), "collection": colIRI, "error": errFw}, "unable to add to collection")
				}
			}
		}
	}
	return it, err
}

// IsLocalIRI shows if the received IRI belongs to the current instance
func (b repo) IsLocalIRI(i as.IRI) bool {
	if _, err := url.Parse(i.String()); err != nil {
		// not an url
		b.errFn(logrus.Fields{
			"IRI": i,
		}, "Invalid url")
		return false
	}
	return strings.Contains(i.String(), b.baseURL)
}

func addToCollection(db *bolt.DB, rootBkt []byte, iri as.IRI, it as.Item) error {
	if it == nil {
		return errors.Newf("unable to add nil element to collection")
	}
	if len(iri) == 0 {
		return errors.Newf("unable to find collection")
	}
	if len(it.GetLink()) == 0 {
		return errors.Newf("Invalid create collection does not have a valid IRI")
	}
	url, err := iri.URL()
	if err != nil {
		return errors.Annotatef(err, "invalid IRI")
	}
	path := url.Path

	return db.Update(func(tx *bolt.Tx) error {
		var col string
		root := tx.Bucket(rootBkt)
		if root == nil {
			return errors.Errorf("Invalid bucket %s", rootBkt)
		}
		if !root.Writable() {
			return errors.Errorf("Non writeable bucket %s", rootBkt)
		}
		var b *bolt.Bucket
		b, col, err = descendInBucket(root, path)
		if err != nil {
			return errors.Newf("Unable to find %s in root bucket", path)
		}
		if !b.Writable() {
			return errors.Errorf("Non writeable bucket %s", path)
		}
		var iris []as.IRI
		raw := b.Get([]byte(col))
		if len(raw) > 0 {
			err := jsonld.Unmarshal(raw, iris)
			if err != nil {
				return errors.Newf("Unable to unmarshal entries in collection %s", path)
			}
		}
		iris = append(iris, iri)
		raw, err := jsonld.Marshal(iris)
		if err != nil {
			return errors.Newf("Unable to marshal entries in collection %s", path)
		}
		err = b.Put([]byte(col), raw)
		if err != nil {
			return errors.Newf("Unable to save entries to collection %s", path)
		}

		return err
	})
}

// UpdateObject
func (b *repo) UpdateObject(it as.Item) (as.Item, error) {
	return b.SaveObject(it)
}

// DeleteObject
func (b *repo) DeleteObject(it as.Item) (as.Item, error) {
	var err error
	err = b.Open()
	if err != nil {
		return it, err
	}
	defer b.Close()
	var bucket string
	if as.ActivityTypes.Contains(it.GetType()) {
		bucket = bucketActivities
	} else if as.ActorTypes.Contains(it.GetType()) {
		bucket = bucketActors
	} else {
		bucket = bucketObjects
	}
	if it, err = delete(b.d, b.root, it); err == nil {
		b.logFn(nil, "Added new %s: %s", bucket[:len(bucket)-1], it.GetLink())
	}
	return it, err
}

// GenerateID
func (b *repo) GenerateID(it as.Item, partOf as.IRI, by as.Item) (as.ObjectID, error) {
	return ap.GenerateID(it, partOf, by)
}

func (b *repo) Open() error {
	var err error
	b.d, err = bolt.Open(b.path, 0600, nil)
	if err != nil {
		return errors.Annotatef(err, "could not open db")
	}
	err = b.d.Update(func(tx *bolt.Tx) error {
		root := tx.Bucket(b.root)
		if root == nil {
			return errors.NotFoundf("root bucket %s not found", b.root)
		}
		if !root.Writable() {
			return errors.NotFoundf("root bucket %s is not writeable", b.root)
		}
		return nil
	})
	if err != nil {
		return errors.Annotatef(err, "db doesn't contain the correct bucket structure")
	}
	return nil
}

// Close closes the boltdb database if possible.
func (b *repo) Close() error {
	return b.d.Close()
}
