//go:build storage_boltdb || storage_all || (!storage_pgx && !storage_fs && !storage_badger && !storage_sqlite)

package boltdb

import (
	"bytes"
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"git.sr.ht/~mariusor/lw"
	"os"
	"path"
	"path/filepath"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/storage"
	"github.com/go-ap/jsonld"
	"github.com/go-ap/processing"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/crypto/bcrypt"
)

var encodeFn = jsonld.Marshal
var decodeFn = jsonld.Unmarshal

type repo struct {
	d       *bolt.DB
	baseURL string
	root    []byte
	path    string
	logFn   loggerFn
	errFn   loggerFn
}

type loggerFn func(lw.Ctx, string, ...interface{})

const (
	rootBucket       = ":"
	bucketActors     = ap.ActorsType
	bucketActivities = ap.ActivitiesType
	bucketObjects    = ap.ObjectsType
)

// Config
type Config struct {
	Path    string
	BaseURL string
	LogFn   loggerFn
	ErrFn   loggerFn
}

var emptyLogFn = func(lw.Ctx, string, ...interface{}) {}

// New returns a new repo repository
func New(c Config) (*repo, error) {
	p, err := Path(c)
	if err != nil {
		return nil, err
	}
	b := repo{
		root:    []byte(rootBucket),
		path:    p,
		baseURL: c.BaseURL,
		logFn:   emptyLogFn,
		errFn:   emptyLogFn,
	}
	if c.ErrFn != nil {
		b.errFn = c.ErrFn
	}
	if c.LogFn != nil {
		b.logFn = c.LogFn
	}
	return &b, nil
}

func loadItem(raw []byte) (vocab.Item, error) {
	if raw == nil || len(raw) == 0 {
		// TODO(marius): log this instead of stopping the iteration and returning an error
		return nil, errors.Errorf("empty raw item")
	}
	return vocab.UnmarshalJSON(raw)
}

func (r *repo) loadItem(b *bolt.Bucket, key []byte, f processing.Filterable) (vocab.Item, error) {
	// we have found an item
	if len(key) == 0 {
		key = []byte(objectKey)
	}
	raw := b.Get(key)
	if raw == nil {
		return nil, nil
	}
	it, err := loadItem(raw)
	if err != nil {
		return nil, err
	}
	if vocab.IsNil(it) {
		return nil, errors.NotFoundf("not found")
	}
	if it.IsCollection() {
		// we need to dereference them, so no further filtering/processing is needed here
		return it, nil
	}
	if vocab.IsIRI(it) {
		if it, _ = r.loadOneFromBucket(it.GetLink()); vocab.IsNil(it) {
			return nil, errors.NotFoundf("not found")
		}
	}
	typ := it.GetType()
	if vocab.ActorTypes.Contains(typ) {
		vocab.OnActor(it, loadFilteredPropsForActor(r, f))
	}
	if vocab.ObjectTypes.Contains(typ) {
		vocab.OnObject(it, loadFilteredPropsForObject(r, f))
	}
	if vocab.IntransitiveActivityTypes.Contains(typ) {
		vocab.OnIntransitiveActivity(it, loadFilteredPropsForIntransitiveActivity(r, f))
	}
	if vocab.ActivityTypes.Contains(typ) {
		vocab.OnActivity(it, loadFilteredPropsForActivity(r, f))
	}
	if f != nil {
		return ap.FilterIt(it, f)
	}
	return it, nil
}

func loadFilteredPropsForActor(r *repo, f processing.Filterable) func(a *vocab.Actor) error {
	return func(a *vocab.Actor) error {
		return vocab.OnObject(a, loadFilteredPropsForObject(r, f))
	}
}

func loadFilteredPropsForObject(r *repo, f processing.Filterable) func(o *vocab.Object) error {
	return func(o *vocab.Object) error {
		if len(o.Tag) == 0 {
			return nil
		}
		return vocab.OnItemCollection(o.Tag, func(col *vocab.ItemCollection) error {
			for i, t := range *col {
				if vocab.IsNil(t) || !vocab.IsIRI(t) {
					return nil
				}
				if ob, err := r.loadOneFromBucket(t.GetLink()); err == nil {
					(*col)[i] = ob
				}
			}
			return nil
		})
	}
}

func loadFilteredPropsForActivity(r *repo, f processing.Filterable) func(a *vocab.Activity) error {
	return func(a *vocab.Activity) error {
		if ok, fo := ap.FiltersOnActivityObject(f); ok && !vocab.IsNil(a.Object) && vocab.IsIRI(a.Object) {
			if ob, err := r.loadOneFromBucket(a.Object.GetLink()); err == nil {
				if ob, _ = ap.FilterIt(ob, fo); ob != nil {
					a.Object = ob
				}
			}
		}
		return vocab.OnIntransitiveActivity(a, loadFilteredPropsForIntransitiveActivity(r, f))
	}
}

func loadFilteredPropsForIntransitiveActivity(r *repo, f processing.Filterable) func(a *vocab.IntransitiveActivity) error {
	return func(a *vocab.IntransitiveActivity) error {
		if ok, fa := ap.FiltersOnActivityActor(f); ok && !vocab.IsNil(a.Actor) && vocab.IsIRI(a.Actor) {
			if act, err := r.loadOneFromBucket(a.Actor.GetLink()); err == nil {
				if act, _ = ap.FilterIt(act, fa); act != nil {
					a.Actor = act
				}
			}
		}
		if ok, ft := ap.FiltersOnActivityTarget(f); ok && !vocab.IsNil(a.Target) && vocab.IsIRI(a.Target) {
			if t, err := r.loadOneFromBucket(a.Target.GetLink()); err == nil {
				if t, _ = ap.FilterIt(t, ft); t != nil {
					a.Target = t
				}
			}
		}
		return nil
	}
}

func (r *repo) loadItemsElements(f processing.Filterable, iris ...vocab.Item) (vocab.ItemCollection, error) {
	col := make(vocab.ItemCollection, 0)
	err := r.d.View(func(tx *bolt.Tx) error {
		rb := tx.Bucket(r.root)
		if rb == nil {
			return ErrorInvalidRoot(r.root)
		}
		var err error
		for _, iri := range iris {
			var b *bolt.Bucket
			remainderPath := itemBucketPath(iri.GetLink())
			b, remainderPath, err = descendInBucket(rb, remainderPath, false)
			if err != nil || b == nil {
				continue
			}
			it, err := r.loadItem(b, []byte(objectKey), f)
			if err != nil || vocab.IsNil(it) {
				continue
			}
			col = append(col, it)
		}
		return nil
	})
	return col, err
}

func (r *repo) loadOneFromBucket(f processing.Filterable) (vocab.Item, error) {
	col, err := r.loadFromBucket(f)
	if err != nil {
		return nil, err
	}
	if len(col) == 0 {
		return nil, errors.NotFoundf("nothing found")
	}
	return col.First(), nil
}

func (r *repo) CreateService(service vocab.Service) error {
	var err error
	if err = r.Open(); err != nil {
		return err
	}
	defer r.Close()
	return createService(r.d, service)
}

func (r *repo) iterateInBucket(b *bolt.Bucket, f processing.Filterable) (vocab.ItemCollection, uint, error) {
	if b == nil {
		return nil, 0, errors.Errorf("invalid bucket to load from")
	}
	// try to iterate in the current collection
	isObjectKey := func(k []byte) bool {
		return string(k) == objectKey || string(k) == metaDataKey
	}
	c := b.Cursor()
	if c == nil {
		return nil, 0, errors.Errorf("Invalid bucket cursor")
	}
	col := make(vocab.ItemCollection, 0)
	// if no path was returned from descendIntoBucket we iterate over all keys in the current bucket
	for key, _ := c.First(); key != nil; key, _ = c.Next() {
		ob := b
		// TODO(marius): we need to see if we can avoid iterating in a bucket for both the UUID, and the underlying
		//  __raw object, because currently they both get loaded and we need to use col.Contains to avoid duplication
		//  when loading some of the actor collections.
		if !isObjectKey(key) {
			if ob = b.Bucket(key); ob == nil {
				continue
			}
		}
		it, err := r.loadItem(ob, []byte(objectKey), f)
		if err != nil || vocab.IsNil(it) {
			continue
		}
		if it.IsCollection() {
			vocab.OnCollectionIntf(it, func(c vocab.CollectionInterface) error {
				itCol, err := r.loadItemsElements(f, c.Collection()...)
				if len(itCol) > 0 {
					for _, it := range itCol {
						if col.Contains(it.GetLink()) {
							continue
						}
						col = append(col, it)
					}
				}
				return err
			})
		} else if !col.Contains(it.GetLink()) {
			col = append(col, it)
		}
	}
	return col, uint(len(col)), nil
}

var ErrorInvalidRoot = func(b []byte) error {
	return errors.Errorf("Invalid root bucket %s", b)
}

func (r *repo) loadFromBucket(f processing.Filterable) (vocab.ItemCollection, error) {
	col := make(vocab.ItemCollection, 0)
	err := r.d.View(func(tx *bolt.Tx) error {
		rb := tx.Bucket(r.root)
		if rb == nil {
			return ErrorInvalidRoot(r.root)
		}

		iri := f.GetLink()
		// This is the case where the Filter points to a single AP Object IRI
		// TODO(marius): Ideally this should support the case where we use the IRI to point to a bucket path
		//     and on top of that apply the other filters
		fullPath := itemBucketPath(iri)
		var remainderPath []byte
		create := false
		var err error
		var b *bolt.Bucket
		// Assume bucket exists and has keys
		b, remainderPath, err = descendInBucket(rb, fullPath, create)
		if err != nil {
			return err
		}
		if b == nil {
			return errors.Errorf("Invalid bucket %s", fullPath)
		}
		lst := vocab.CollectionPath(path.Base(string(fullPath)))
		if isStorageCollectionKey(lst) {
			fromBucket, _, err := r.iterateInBucket(b, f)
			if err != nil {
				return err
			}
			col = append(col, fromBucket...)
		} else if len(remainderPath) == 0 {
			// we have found an item
			key := []byte(objectKey)
			it, err := r.loadItem(b, key, f)
			if err != nil {
				return err
			}
			if vocab.IsNil(it) {
				if isStorageCollectionKey(lst) {
					return nil
				}
				return errors.NotFoundf("not found")
			}
			if it.IsCollection() {
				isColFn := func(ff processing.Filterable) bool {
					_, ok := ff.(vocab.IRI)
					return ok
				}
				return vocab.OnCollectionIntf(it, func(c vocab.CollectionInterface) error {
					if isColFn(f) {
						f = c.Collection()
					}
					col, err = r.loadItemsElements(f, c.Collection()...)
					return err
				})
			}
			if !col.Contains(it) {
				col = append(col, it)
			}
			return nil
		}
		return nil
	})

	return col, err
}

func (r repo) buildIRIs(c vocab.CollectionPath, hashes ...ap.Hash) vocab.IRIs {
	iris := make(vocab.IRIs, 0)
	for _, hash := range hashes {
		i := c.IRI(vocab.IRI(r.baseURL)).AddPath(hash.String())
		iris = append(iris, i)
	}
	return iris
}

// Load
func (r *repo) Load(i vocab.IRI) (vocab.Item, error) {
	if err := r.Open(); err != nil {
		return nil, err
	}
	defer r.Close()
	if !r.IsLocalIRI(vocab.IRI(r.baseURL)) {
		return nil, errors.Newf("unable to load non-local IRI: %s", i)
	}

	f, err := ap.FiltersFromIRI(i)
	if err != nil {
		return nil, err
	}

	ret, err := r.loadFromBucket(f)
	if len(ret) == 1 && f.IsItemIRI() {
		return ret.First(), err
	}
	return ret, err
}

func descendInBucket(root *bolt.Bucket, path []byte, create bool) (*bolt.Bucket, []byte, error) {
	if root == nil {
		return nil, path, errors.Newf("trying to descend into nil bucket")
	}
	if len(path) == 0 {
		return root, path, nil
	}
	buckets := bytes.Split(path, []byte{'/'})

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
			cb, _ = b.CreateBucketIfNotExists(name)
		} else {
			cb = b.Bucket(name)
		}
		if cb == nil {
			lvl--
			break
		}
		b = cb
	}
	remBuckets := buckets[lvl:]
	path = bytes.Join(remBuckets, []byte{'/'})
	if len(remBuckets) > 0 && !ap.HiddenCollections.Contains(vocab.CollectionPath(path)) {
		return b, path, errors.NotFoundf("%s not found", remBuckets[0])
	}
	return b, path, nil
}

const objectKey = "__raw"
const metaDataKey = "__meta_data"

func delete(r *repo, it vocab.Item) error {
	if it.IsCollection() {
		return vocab.OnCollectionIntf(it, func(c vocab.CollectionInterface) error {
			var err error
			for _, it := range c.Collection() {
				if err = deleteItem(r, it); err != nil {
					r.logFn(nil, "Unable to remove item %s", it.GetLink())
				}
			}
			return nil
		})
	}

	return deleteItem(r, it.GetLink())
}

// Create
func (r *repo) Create(col vocab.CollectionInterface) (vocab.CollectionInterface, error) {
	var err error
	err = r.Open()
	if err != nil {
		return col, err
	}
	defer r.Close()

	cPath := itemBucketPath(col.GetLink())
	c := []byte(path.Base(string(cPath)))
	err = r.d.Update(func(tx *bolt.Tx) error {
		root, err := tx.CreateBucketIfNotExists(r.root)
		if err != nil {
			return err
		}
		b, _, err := descendInBucket(root, cPath, true)
		if err != nil {
			return errors.Annotatef(err, "Unable to find path %s/%s", r.root, cPath)
		}
		return b.Put(c, nil)
	})
	return col, err
}

func itemBucketPath(iri vocab.IRI) []byte {
	url, err := iri.URL()
	if err != nil {
		return nil
	}
	return []byte(url.Host + url.Path)
}

func createCollectionInBucket(b *bolt.Bucket, it vocab.Item) (vocab.Item, error) {
	if vocab.IsNil(it) {
		return nil, nil
	}
	p := []byte(path.Base(it.GetLink().String()))
	_, err := b.CreateBucketIfNotExists(p)
	if err != nil {
		return nil, err
	}
	return it.GetLink(), nil
}

func deleteBucket(b *bolt.Bucket, it vocab.Item) error {
	if vocab.IsNil(it) {
		return nil
	}
	p := []byte(it.GetLink())
	return b.DeleteBucket(p)
}

func createCollectionsInBucket(b *bolt.Bucket, it vocab.Item) error {
	if vocab.IsNil(it) || !it.IsObject() {
		return nil
	}
	// create collections
	if vocab.ActorTypes.Contains(it.GetType()) {
		vocab.OnActor(it, func(p *vocab.Actor) error {
			if p.Inbox != nil {
				p.Inbox, _ = createCollectionInBucket(b, vocab.Inbox.IRI(p))
			}
			if p.Outbox != nil {
				p.Outbox, _ = createCollectionInBucket(b, vocab.Outbox.IRI(p))
			}
			if p.Followers != nil {
				p.Followers, _ = createCollectionInBucket(b, vocab.Followers.IRI(p))
			}
			if p.Following != nil {
				p.Following, _ = createCollectionInBucket(b, vocab.Liked.IRI(p))
			}
			if p.Liked != nil {
				p.Liked, _ = createCollectionInBucket(b, vocab.Liked.IRI(p))
			}
			return nil
		})
	}
	return vocab.OnObject(it, func(o *vocab.Object) error {
		if o.Replies != nil {
			o.Replies, _ = createCollectionInBucket(b, vocab.Replies.IRI(o))
		}
		if o.Likes != nil {
			o.Likes, _ = createCollectionInBucket(b, vocab.Likes.IRI(o))
		}
		if o.Shares != nil {
			o.Shares, _ = createCollectionInBucket(b, vocab.Shares.IRI(o))
		}
		return nil
	})
}

// deleteItem
func deleteItem(r *repo, it vocab.Item) error {
	pathInBucket := itemBucketPath(it.GetLink())
	return r.d.Update(func(tx *bolt.Tx) error {
		root := tx.Bucket(r.root)
		if root == nil {
			return ErrorInvalidRoot(r.root)
		}
		if !root.Writable() {
			return errors.Errorf("Non writeable bucket %s", r.root)
		}
		b, _, err := descendInBucket(root, pathInBucket, true)
		if err != nil {
			return errors.Annotatef(err, "Unable to find %s in root bucket", pathInBucket)
		}
		if !b.Writable() {
			return errors.Errorf("Non writeable bucket %s", pathInBucket)
		}
		return deleteBucket(b, it)
	})
}

// deleteCollectionsFromBucket
func deleteCollectionsFromBucket(b *bolt.Bucket, it vocab.Item) error {
	if vocab.ActorTypes.Contains(it.GetType()) {
		return vocab.OnActor(it, func(p *vocab.Actor) error {
			var err error
			err = deleteBucket(b, vocab.Inbox.IRI(p))
			err = deleteBucket(b, vocab.Outbox.IRI(p))
			err = deleteBucket(b, vocab.Followers.IRI(p))
			err = deleteBucket(b, vocab.Following.IRI(p))
			err = deleteBucket(b, vocab.Liked.IRI(p))
			return err
		})
	}
	if vocab.ObjectTypes.Contains(it.GetType()) {
		return vocab.OnObject(it, func(o *vocab.Object) error {
			var err error
			err = deleteBucket(b, vocab.Replies.IRI(o))
			err = deleteBucket(b, vocab.Likes.IRI(o))
			err = deleteBucket(b, vocab.Shares.IRI(o))
			return err
		})
	}
	return nil
}
func save(r *repo, it vocab.Item) (vocab.Item, error) {
	pathInBucket := itemBucketPath(it.GetLink())
	err := r.d.Update(func(tx *bolt.Tx) error {
		root, err := tx.CreateBucketIfNotExists(r.root)
		if err != nil {
			return errors.Errorf("Not able to write to root bucket %s", r.root)
		}
		if root == nil {
			return ErrorInvalidRoot(r.root)
		}
		if !root.Writable() {
			return errors.Errorf("Non writeable bucket %s", r.root)
		}
		b, uuid, err := descendInBucket(root, pathInBucket, true)
		if err != nil {
			return errors.Annotatef(err, "Unable to find %s in root bucket", pathInBucket)
		}
		if !b.Writable() {
			return errors.Errorf("Non writeable bucket %s", pathInBucket)
		}
		if len(uuid) == 0 {
			if err := createCollectionsInBucket(b, it); err != nil {
				return errors.Annotatef(err, "could not create object's collections")
			}
		}

		// TODO(marius): it's possible to set the encoding/decoding functions on the package or storage object level
		//  instead of using jsonld.(Un)Marshal like this.
		entryBytes, err := encodeFn(it)
		if err != nil {
			return errors.Annotatef(err, "could not marshal object")
		}
		err = b.Put([]byte(objectKey), entryBytes)
		if err != nil {
			return errors.Annotatef(err, "could not store encoded object")
		}

		return nil
	})

	return it, err
}

// Save
func (r *repo) Save(it vocab.Item) (vocab.Item, error) {
	var err error
	err = r.Open()
	if err != nil {
		return it, err
	}
	defer r.Close()

	if it, err = save(r, it); err == nil {
		op := "Updated"
		id := it.GetID()
		if !id.IsValid() {
			op = "Added new"
		}
		r.logFn(nil, "%s %s: %s", op, it.GetType(), it.GetLink())
	}

	return it, err
}

// IsLocalIRI shows if the received IRI belongs to the current instance
func (r repo) IsLocalIRI(i vocab.IRI) bool {
	return i.Contains(vocab.IRI(r.baseURL), false)
}

func onCollection(r *repo, col vocab.IRI, it vocab.Item, fn func(iris vocab.IRIs) (vocab.IRIs, error)) error {
	if vocab.IsNil(it) {
		return errors.Newf("Unable to operate on nil element")
	}
	if len(col) == 0 {
		return errors.Newf("Unable to find collection")
	}
	if len(it.GetLink()) == 0 {
		return errors.Newf("Invalid collection, it does not have a valid IRI")
	}
	if !r.IsLocalIRI(col.GetLink()) {
		return errors.Newf("Unable to save to non local collection %s", col)
	}
	path := itemBucketPath(col.GetLink())
	err := r.Open()
	if err != nil {
		return err
	}
	defer r.Close()

	return r.d.Update(func(tx *bolt.Tx) error {
		var rem []byte
		root := tx.Bucket(r.root)
		if root == nil {
			return ErrorInvalidRoot(r.root)
		}
		if !root.Writable() {
			return errors.Errorf("Non writeable bucket %s", r.root)
		}
		var b *bolt.Bucket
		b, rem, err = descendInBucket(root, path, true)
		if err != nil {
			return errors.Newf("Unable to find %s in root bucket", path)
		}
		if len(rem) == 0 {
			rem = []byte(objectKey)
		}
		if !b.Writable() {
			return errors.Errorf("Non writeable bucket %s", path)
		}
		var iris vocab.IRIs
		raw := b.Get(rem)
		if len(raw) > 0 {
			err := decodeFn(raw, &iris)
			if err != nil {
				return errors.Newf("Unable to unmarshal entries in collection %s", path)
			}
		}
		iris, err = fn(iris)
		if err != nil {
			return errors.Annotatef(err, "Unable operate on collection %s", path)
		}
		raw, err = encodeFn(iris)
		if err != nil {
			return errors.Newf("Unable to marshal entries in collection %s", path)
		}
		err = b.Put(rem, raw)
		if err != nil {
			return errors.Newf("Unable to save entries to collection %s", path)
		}
		return err
	})
}

// RemoveFrom
func (r *repo) RemoveFrom(col vocab.IRI, it vocab.Item) error {
	return onCollection(r, col, it, func(iris vocab.IRIs) (vocab.IRIs, error) {
		for k, iri := range iris {
			if iri.GetLink().Equals(it.GetLink(), false) {
				iris = append(iris[:k], iris[k+1:]...)
				break
			}
		}
		return iris, nil
	})
}

func isStorageCollectionKey(lst vocab.CollectionPath) bool {
	return ap.FedBOXCollections.Contains(lst) || vocab.OfActor.Contains(lst) || vocab.OfObject.Contains(lst)
}

func addCollectionOnObject(r *repo, col vocab.IRI) error {
	var err error
	allStorageCollections := append(vocab.ActivityPubCollections, ap.FedBOXCollections...)
	if ob, t := allStorageCollections.Split(col); vocab.ValidCollection(t) {
		// Create the collection on the object, if it doesn't exist
		i, _ := r.loadOneFromBucket(ob)
		if _, ok := t.AddTo(i); ok {
			_, err = r.Save(i)
		}
	}
	return err
}

// AddTo
func (r *repo) AddTo(col vocab.IRI, it vocab.Item) error {
	addCollectionOnObject(r, col)
	return onCollection(r, col, it, func(iris vocab.IRIs) (vocab.IRIs, error) {
		if iris.Contains(it.GetLink()) {
			return iris, nil
		}
		return append(iris, it.GetLink()), nil
	})
}

// Delete
func (r *repo) Delete(it vocab.Item) error {
	err := r.Open()
	if err != nil {
		return err
	}
	defer r.Close()
	return delete(r, it)
}

// Open opens the boltdb database if possible.
func (r *repo) Open() error {
	if r == nil {
		return errors.Newf("Unable to open uninitialized db")
	}
	var err error
	r.d, err = bolt.Open(r.path, 0600, nil)
	if err != nil {
		return errors.Annotatef(err, "Could not open db %s", r.path)
	}
	return nil
}

// Close closes the boltdb database if possible.
func (r *repo) Close() error {
	if r == nil {
		return errors.Newf("Unable to close uninitialized db")
	}
	if r.d == nil {
		return nil
	}
	return r.d.Close()
}

// PasswordSet
func (r *repo) PasswordSet(it vocab.Item, pw []byte) error {
	path := itemBucketPath(it.GetLink())
	err := r.Open()
	if err != nil {
		return err
	}
	defer r.Close()

	err = r.d.Update(func(tx *bolt.Tx) error {
		root, err := tx.CreateBucketIfNotExists(r.root)
		if err != nil {
			return errors.Errorf("Not able to write to root bucket %s", r.root)
		}
		if root == nil {
			return ErrorInvalidRoot(r.root)
		}
		if !root.Writable() {
			return errors.Errorf("Non writeable bucket %s", r.root)
		}
		var b *bolt.Bucket
		b, _, err = descendInBucket(root, path, true)
		if err != nil {
			return errors.Newf("Unable to find %s in root bucket", path)
		}
		if !b.Writable() {
			return errors.Errorf("Non writeable bucket %s", path)
		}

		pw, err = bcrypt.GenerateFromPassword(pw, -1)
		if err != nil {
			return errors.Annotatef(err, "Could not encrypt the pw")
		}
		m := storage.Metadata{
			Pw: pw,
		}
		entryBytes, err := encodeFn(m)
		if err != nil {
			return errors.Annotatef(err, "Could not marshal metadata")
		}
		err = b.Put([]byte(metaDataKey), entryBytes)
		if err != nil {
			return errors.Errorf("Could not insert entry: %s", err)
		}
		return nil
	})

	return err
}

// PasswordCheck
func (r *repo) PasswordCheck(it vocab.Item, pw []byte) error {
	path := itemBucketPath(it.GetLink())
	err := r.Open()
	if err != nil {
		return err
	}
	defer r.Close()

	m := storage.Metadata{}
	err = r.d.View(func(tx *bolt.Tx) error {
		root := tx.Bucket(r.root)
		if root == nil {
			return ErrorInvalidRoot(r.root)
		}
		var b *bolt.Bucket
		b, path, err = descendInBucket(root, path, false)
		if err != nil {
			return errors.Newf("Unable to find %s in root bucket", path)
		}
		entryBytes := b.Get([]byte(metaDataKey))
		err := decodeFn(entryBytes, &m)
		if err != nil {
			return errors.Annotatef(err, "Could not unmarshal metadata")
		}
		if err := bcrypt.CompareHashAndPassword(m.Pw, pw); err != nil {
			return errors.NewUnauthorized(err, "Invalid pw")
		}
		return nil
	})
	return err
}

// LoadMetadata
func (r *repo) LoadMetadata(iri vocab.IRI) (*storage.Metadata, error) {
	err := r.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	path := itemBucketPath(iri)

	var m *storage.Metadata
	err = r.d.View(func(tx *bolt.Tx) error {
		root := tx.Bucket(r.root)
		if root == nil {
			return ErrorInvalidRoot(r.root)
		}
		var b *bolt.Bucket
		b, path, err = descendInBucket(root, path, false)
		if err != nil {
			return errors.Newf("Unable to find %s in root bucket", path)
		}
		entryBytes := b.Get([]byte(metaDataKey))
		m = new(storage.Metadata)
		return decodeFn(entryBytes, m)
	})
	return m, err
}

// SaveMetadata
func (r *repo) SaveMetadata(m storage.Metadata, iri vocab.IRI) error {
	err := r.Open()
	if err != nil {
		return err
	}
	defer r.Close()

	path := itemBucketPath(iri)
	err = r.d.Update(func(tx *bolt.Tx) error {
		root, err := tx.CreateBucketIfNotExists(r.root)
		if err != nil {
			return errors.Errorf("Not able to write to root bucket %s", r.root)
		}
		if root == nil {
			return ErrorInvalidRoot(r.root)
		}
		if !root.Writable() {
			return errors.Errorf("Non writeable bucket %s", r.root)
		}
		var b *bolt.Bucket
		b, _, err = descendInBucket(root, path, true)
		if err != nil {
			return errors.Newf("Unable to find %s in root bucket", path)
		}
		if !b.Writable() {
			return errors.Errorf("Non writeable bucket %s", path)
		}

		entryBytes, err := encodeFn(m)
		if err != nil {
			return errors.Annotatef(err, "Could not marshal metadata")
		}
		err = b.Put([]byte(metaDataKey), entryBytes)
		if err != nil {
			return errors.Errorf("Could not insert entry: %s", err)
		}
		return nil
	})

	return err
}

// LoadKey loads a private key for an actor found by its IRI
func (r *repo) LoadKey(iri vocab.IRI) (crypto.PrivateKey, error) {
	m, err := r.LoadMetadata(iri)
	if err != nil {
		return nil, err
	}
	b, _ := pem.Decode(m.PrivateKey)
	if b == nil {
		return nil, errors.Errorf("failed decoding pem")
	}
	prvKey, err := x509.ParsePKCS8PrivateKey(b.Bytes)
	if err != nil {
		return nil, err
	}
	return prvKey, nil
}

func Path(c Config) (string, error) {
	if !filepath.IsAbs(c.Path) {
		c.Path, _ = filepath.Abs(c.Path)
	}
	if err := mkDirIfNotExists(c.Path); err != nil {
		return "", err
	}
	p := path.Join(c.Path, "storage.bdb")
	return p, nil
}

func mkDirIfNotExists(p string) error {
	fi, err := os.Stat(p)
	if err != nil && os.IsNotExist(err) {
		err = os.MkdirAll(p, os.ModeDir|os.ModePerm|0700)
	}
	if err != nil {
		return err
	}
	fi, err = os.Stat(p)
	if err != nil {
		return err
	} else if !fi.IsDir() {
		return errors.Errorf("path exists, and is not a folder %s", p)
	}
	return nil
}
