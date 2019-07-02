package boltdb

import (
	"bytes"
	"fmt"
	"github.com/boltdb/bolt"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/jsonld"
	s "github.com/go-ap/storage"
	"github.com/sirupsen/logrus"
	"net/url"
	"path"
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
		iri := f.GetLink()
		url, err := iri.URL()
		if err != nil {
			return errors.Newf("invalid IRI filter element %s when loading collections", iri)
		}
		if string(root) != url.Host {
			return errors.Newf("trying to load from non-local root bucket %s", url.Host)
		}
		// Assume bucket exists and has keys
		b, path, err := descendInBucket(rb, url.Path, false)
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
func (r *repo) Load(f s.Filterable) (as.ItemCollection, uint, error) {
	var err error
	err = r.Open()
	if err != nil {
		return nil, 0, err
	}
	defer r.Close()
	return nil, 0, errors.NotImplementedf("BoltDB Load not implemented")
}

// LoadActivities
func (r *repo) LoadActivities(f s.Filterable) (as.ItemCollection, uint, error) {
	var err error
	err = r.Open()
	if err != nil {
		return nil, 0, err
	}
	defer r.Close()
	return loadFromBucket(r.d, r.root, f)
}

// LoadObjects
func (r *repo) LoadObjects(f s.Filterable) (as.ItemCollection, uint, error) {
	var err error
	err = r.Open()
	if err != nil {
		return nil, 0, err
	}
	defer r.Close()
	return loadFromBucket(r.d, r.root, f)
}

// LoadActors
func (r *repo) LoadActors(f s.Filterable) (as.ItemCollection, uint, error) {
	var err error
	err = r.Open()
	if err != nil {
		return nil, 0, err
	}
	defer r.Close()
	return loadFromBucket(r.d, r.root, f)
}

func descendInBucket(root *bolt.Bucket, path string, create bool) (*bolt.Bucket, string, error) {
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
		var cb *bolt.Bucket
		if create {
			cb, _ = b.CreateBucketIfNotExists([]byte(name))
		} else {
			cb = b.Bucket([]byte(name))
		}
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
func (r *repo) LoadCollection(f s.Filterable) (as.CollectionInterface, error) {
	var err error
	err = r.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var ret as.CollectionInterface
	iri := f.GetLink()
	url, err := iri.URL()
	if err != nil {
		r.errFn(nil, "invalid IRI filter element %s when loading collections", iri)
	}
	if string(r.root) != url.Host {
		r.errFn(nil, "trying to load from non-local root bucket %s", url.Host)
	}
	col := &as.OrderedCollection{}
	col.ID = as.ObjectID(iri)
	col.Type = as.OrderedCollectionType

	err = r.d.View(func(tx *bolt.Tx) error {
		rb := tx.Bucket(r.root)
		if rb == nil {
			return errors.Newf("invalid root bucket %s", r.root)
		}
		bb, path, err := descendInBucket(rb, url.Path, false)
		if err != nil {
			r.errFn(nil, "unable to find %s in root bucket", path, r.root)
		}
		if len(path) == 0 {
			cb := bb.Cursor()
			if cb == nil {
				return errors.Errorf("Invalid collection bucket path %s", path)
			}
			for uuid, _ := cb.First(); uuid != nil; uuid, _ = cb.Next() {
				ib := bb.Bucket(uuid)
				if ib == nil {
					return nil
				}
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
			return errors.NotImplementedf("TODO: unmarshal collection items in collection: %s", url.Path)
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

func delete(r *repo, it as.Item) (as.Item, error) {
	f := ap.Filters{
		IRI: it.GetLink(),
	}
	if it.IsObject() {
		f.Type = []as.ActivityVocabularyType{it.GetType()}
	}
	var cnt uint
	var found as.ItemCollection
	found, cnt, _ = loadFromBucket(r.d, r.root, &f)
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
				ap.ActivityStreamsPublicNS,
			},
		},
		Deleted:    time.Now().UTC(),
		FormerType: old.GetType(),
	}
	return save(r, t)
}

func (r *repo) CreateCollection(col as.CollectionInterface) (as.CollectionInterface, error) {
	var err error
	err = r.Open()
	if err != nil {
		return col, err
	}
	defer r.Close()

	url, err := col.GetLink().URL()
	if err != nil {
		return col, err
	}
	cPath := url.Path
	c := path.Base(cPath)
	err = r.d.Update(func(tx *bolt.Tx) error {
		root := tx.Bucket(r.root)
		b, _, err := descendInBucket(root, cPath, true)
		if err != nil {
			return err
		}
		return b.Put([]byte(c), nil)
	})
	return col, err
}

func save(r *repo, it as.Item) (as.Item, error) {
	url, err := it.GetLink().URL()
	if err != nil {
		return it, errors.Annotatef(err, "invalid IRI")
	}
	path := url.Path

	var uuid string
	err = r.d.Update(func(tx *bolt.Tx) error {
		root := tx.Bucket(r.root)
		if root == nil {
			return errors.Errorf("Invalid bucket %s", r.root)
		}
		if !root.Writable() {
			return errors.Errorf("Non writeable bucket %s", r.root)
		}
		var b *bolt.Bucket
		b, uuid, err = descendInBucket(root, path, true)
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
		return nil
	})

	err = r.d.Update(func(tx *bolt.Tx) error {
		root := tx.Bucket(r.root)
		if root == nil {
			return errors.Errorf("Invalid bucket %s", r.root)
		}
		if !root.Writable() {
			return errors.Errorf("Non writeable bucket %s", r.root)
		}
		var b *bolt.Bucket
		b, uuid, err = descendInBucket(root, path, true)
		if err != nil {
			return errors.Newf("Unable to find %s in root bucket", path)
		}
		if !b.Writable() {
			return errors.Errorf("Non writeable bucket %s", path)
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
func (r *repo) SaveActivity(it as.Item) (as.Item, error) {
	var err error
	if it, err = r.SaveObject(it); err == nil {
		r.logFn(nil, "Added new activity: %s", it.GetLink())
	}
	return it, err
}

// SaveObject
func (r *repo) SaveObject(it as.Item) (as.Item, error) {
	var err error
	err = r.Open()
	if err != nil {
		return it, err
	}
	defer r.Close()

	if it, err = save(r, it); err == nil {
		r.logFn(nil, "Added new %s: %s", it.GetType(), it.GetLink())
	}

	// TODO(marius) Move to somewhere else
	if toFw, ok := it.(as.HasRecipients); ok {
		for _, fw := range toFw.Recipients() {
			colIRI := fw.GetLink()
			if r.IsLocalIRI(colIRI) {
				// we shadow the err variable intentionally so it does not propagate upper to the call stack
				if errFw := r.AddToCollection(colIRI, it); err != nil {
					r.errFn(logrus.Fields{"IRI": it.GetLink(), "collection": colIRI, "error": errFw}, "unable to add to collection")
				}
			}
		}
	}
	return it, err
}

// IsLocalIRI shows if the received IRI belongs to the current instance
func (r repo) IsLocalIRI(i as.IRI) bool {
	if _, err := url.Parse(i.String()); err != nil {
		// not an url
		r.errFn(logrus.Fields{
			"IRI": i,
		}, "Invalid url")
		return false
	}
	return strings.Contains(i.String(), r.baseURL)
}

func (r *repo) AddToCollection(col as.IRI, it as.Item) error {
	if it == nil {
		return errors.Newf("unable to add nil element to collection")
	}
	if len(col) == 0 {
		return errors.Newf("unable to find collection")
	}
	if len(it.GetLink()) == 0 {
		return errors.Newf("Invalid create collection does not have a valid IRI")
	}
	url, err := col.URL()
	if err != nil {
		return errors.Annotatef(err, "invalid IRI")
	}
	path := url.Path

	return r.d.Update(func(tx *bolt.Tx) error {
		var rem string
		root := tx.Bucket(r.root)
		if root == nil {
			return errors.Errorf("Invalid bucket %s", r.root)
		}
		if !root.Writable() {
			return errors.Errorf("Non writeable bucket %s", r.root)
		}
		var b *bolt.Bucket
		b, rem, err = descendInBucket(root, path, true)
		if err != nil {
			return errors.Newf("Unable to find %s in root bucket", path)
		}
		if !b.Writable() {
			return errors.Errorf("Non writeable bucket %s", path)
		}
		var iris []as.IRI
		raw := b.Get([]byte(rem))
		if len(raw) > 0 {
			err := jsonld.Unmarshal(raw, iris)
			if err != nil {
				return errors.Newf("Unable to unmarshal entries in collection %s", path)
			}
		}
		iris = append(iris, col)
		raw, err := jsonld.Marshal(iris)
		if err != nil {
			return errors.Newf("Unable to marshal entries in collection %s", path)
		}
		err = b.Put([]byte(rem), raw)
		if err != nil {
			return errors.Newf("Unable to save entries to collection %s", path)
		}

		return err
	})
}

// UpdateObject
func (r *repo) UpdateObject(it as.Item) (as.Item, error) {
	return r.SaveObject(it)
}

// DeleteObject
func (r *repo) DeleteObject(it as.Item) (as.Item, error) {
	var err error
	err = r.Open()
	if err != nil {
		return it, err
	}
	defer r.Close()
	var bucket string
	if as.ActivityTypes.Contains(it.GetType()) {
		bucket = bucketActivities
	} else if as.ActorTypes.Contains(it.GetType()) {
		bucket = bucketActors
	} else {
		bucket = bucketObjects
	}
	if it, err = delete(r, it); err == nil {
		r.logFn(nil, "Added new %s: %s", bucket[:len(bucket)-1], it.GetLink())
	}
	return it, err
}

// GenerateID
func (r *repo) GenerateID(it as.Item, by as.Item) (as.ObjectID, error) {
	typ := it.GetType()
	var partOf string
	if as.ActivityTypes.Contains(typ) {
		partOf = fmt.Sprintf("%s/activities", r.baseURL)
	} else if as.ActorTypes.Contains(typ) {
		partOf = fmt.Sprintf("%s/actors", r.baseURL)
	} else if as.ObjectTypes.Contains(typ) {
		partOf = fmt.Sprintf("%s/objects", r.baseURL)
	}
	return ap.GenerateID(it, partOf, by)
}

func (r *repo) Open() error {
	var err error
	r.d, err = bolt.Open(r.path, 0600, nil)
	if err != nil {
		return errors.Annotatef(err, "could not open db")
	}
	err = r.d.Update(func(tx *bolt.Tx) error {
		root := tx.Bucket(r.root)
		if root == nil {
			return errors.NotFoundf("root bucket %s not found", r.root)
		}
		if !root.Writable() {
			return errors.NotFoundf("root bucket %s is not writeable", r.root)
		}
		return nil
	})
	if err != nil {
		return errors.Annotatef(err, "db doesn't contain the correct bucket structure")
	}
	return nil
}

// Close closes the boltdb database if possible.
func (r *repo) Close() error {
	return r.d.Close()
}
