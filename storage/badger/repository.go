// +build storage_badger storage_all

package badger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/dgraph-io/badger/v2"
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/storage"
	"github.com/go-ap/handlers"
	"github.com/go-ap/jsonld"
	s "github.com/go-ap/storage"
	"github.com/mariusor/qstring"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"path"
	"sort"
	"sync"
	"time"
)

const (
	pathActors     = ap.ActorsType
	pathActivities = ap.ActivitiesType
	pathObjects    = ap.ObjectsType
)

type repo struct {
	d       *badger.DB
	m       sync.Mutex
	baseURL string
	path    string
	logFn   loggerFn
	errFn   loggerFn
}

type loggerFn func(logrus.Fields, string, ...interface{})

// Config
type Config struct {
	Path     string
	PathName string
	LogFn    loggerFn
	ErrFn    loggerFn
}

var emptyLogFn = func(logrus.Fields, string, ...interface{}) {}

// New returns a new repo repository
func New(c Config, baseURL string) *repo {
	b := repo{
		path:    c.Path,
		m:       sync.Mutex{},
		baseURL: baseURL,
		logFn:   emptyLogFn,
		errFn:   emptyLogFn,
	}
	if c.ErrFn != nil {
		b.errFn = c.ErrFn
	}
	if c.LogFn != nil {
		b.logFn = c.LogFn
	}
	return &b
}

// Open opens the badger database if possible.
func (r *repo) Open() error {
	var err error
	r.m.Lock()
	c := badger.DefaultOptions(r.path).WithLogger(logger{
		logFn: r.logFn,
		errFn: r.errFn,
	})
	r.d, err = badger.Open(c)
	if err != nil {
		err = errors.Annotatef(err, "unable to open storage")
	}
	return err
}

// Close closes the badger database if possible.
func (r *repo) Close() error {
	if r.d == nil {
		return nil
	}
	err := r.d.Close()
	r.m.Unlock()
	return err
}

// Load
func (r *repo) Load(f s.Filterable) (pub.ItemCollection, uint, error) {
	var err error
	if r.Open(); err != nil {
		return nil, 0, err
	}
	defer r.Close()

	return r.loadFromPath(f)
}

// LoadActivities
func (r *repo) LoadActivities(f s.Filterable) (pub.ItemCollection, uint, error) {
	return r.Load(f)
}

// LoadObjects
func (r *repo) LoadObjects(f s.Filterable) (pub.ItemCollection, uint, error) {
	return r.Load(f)
}

// LoadActors
func (r *repo) LoadActors(f s.Filterable) (pub.ItemCollection, uint, error) {
	return r.Load(f)
}

func orderItems(col pub.ItemCollection) pub.ItemCollection {
	sort.SliceStable(col, func(i, j int) bool {
		return pub.ItemOrderTimestamp(col[i], col[j])
	})
	return col
}

// LoadCollection
func (r *repo) LoadCollection(f s.Filterable) (pub.CollectionInterface, error) {
	var err error
	err = r.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()

	iri := f.GetLink()
	url, err := iri.URL()
	if err != nil {
		r.errFn(nil, "invalid IRI filter element %s when loading collections", iri)
	}

	q, _ := qstring.Marshal(f)
	url.RawQuery = q.Encode()

	col := &pub.OrderedCollection{}
	col.ID = pub.ID(url.String())
	col.Type = pub.OrderedCollectionType

	elements, count, err := r.loadFromPath(f)
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return col, nil
	}
	col.OrderedItems = orderItems(elements)
	col.TotalItems = count

	return col, err
}

func (r *repo) CreateCollection(col pub.CollectionInterface) (pub.CollectionInterface, error) {
	var err error
	err = r.Open()
	if err != nil {
		return col, err
	}
	defer r.Close()

	cPath := itemPath(col.GetLink())
	c := []byte(path.Base(string(cPath)))
	err = r.d.Update(func(tx *badger.Txn) error {
		return tx.Set(c, nil)
	})
	return col, err
}

// SaveActivity
func (r *repo) SaveActivity(it pub.Item) (pub.Item, error) {
	return r.SaveObject(it)
}

// SaveActor
func (r *repo) SaveActor(it pub.Item) (pub.Item, error) {
	return r.SaveObject(it)
}

// SaveObject
func (r *repo) SaveObject(it pub.Item) (pub.Item, error) {
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
func (r repo) IsLocalIRI(i pub.IRI) bool {
	return i.Contains(pub.IRI(r.baseURL), false)
}
func getCollection(it pub.Item, c handlers.CollectionType) pub.CollectionInterface {
	return &pub.OrderedCollection{
		ID:   c.IRI(it).GetLink(),
		Type: pub.OrderedCollectionType,
	}
}

func addNewObjectCollections(it pub.Item) (pub.Item, error) {
	if pub.ActorTypes.Contains(it.GetType()) {
		pub.OnActor(it, func(p *pub.Actor) error {
			p.Inbox = getCollection(p, handlers.Inbox)
			p.Outbox = getCollection(p, handlers.Outbox)
			p.Followers = getCollection(p, handlers.Followers)
			p.Following = getCollection(p, handlers.Following)
			p.Liked = getCollection(p, handlers.Liked)
			return nil
		})
	}
	if pub.ObjectTypes.Contains(it.GetType()) {
		pub.OnObject(it, func(o *pub.Object) error {
			o.Replies = getCollection(o, handlers.Replies)
			o.Likes = getCollection(o, handlers.Likes)
			o.Shares = getCollection(o, handlers.Shares)
			return nil
		})
	}
	return it, nil
}
func onCollection(r *repo, col pub.IRI, it pub.Item, fn func(iris pub.IRIs) (pub.IRIs, error)) error {
	if it == nil {
		return errors.Newf("Unable to operate on nil element")
	}
	if len(col) == 0 {
		return errors.Newf("Unable to find collection")
	}
	if len(it.GetLink()) == 0 {
		return errors.Newf("Invalid collection, it does not have a valid IRI")
	}
	if !r.IsLocalIRI(col) {
		return errors.Newf("Unable to save to non local collection %s", col)
	}
	p := itemPath(col)

	err := r.Open()
	if err != nil {
		return err
	}
	defer r.Close()
	return r.d.Update(func(tx *badger.Txn) error {
		iris := make(pub.IRIs, 0)

		rawKey := getObjectKey(p)
		if i, err := tx.Get(rawKey); err == nil {
			err = i.Value(func(raw []byte) error {
				err := jsonld.Unmarshal(raw, &iris)
				if err != nil {
					return errors.Annotatef(err, "Unable to unmarshal collection %s", p)
				}
				return nil
			})
		}
		var err error
		iris, err = fn(iris)
		if err != nil {
			return errors.Annotatef(err, "Unable operate on collection %s", p)
		}
		var raw []byte
		raw, err = jsonld.Marshal(iris)
		if err != nil {
			return errors.Newf("Unable to marshal entries in collection %s", p)
		}
		err = tx.Set(rawKey, raw)
		if err != nil {
			return errors.Annotatef(err, "Unable to save entries to collection %s", p)
		}
		return err
	})
}

// RemoveFromCollection
func (r *repo) RemoveFromCollection(col pub.IRI, it pub.Item) error {
	return onCollection(r, col, it, func(iris pub.IRIs) (pub.IRIs, error) {
		for k, iri := range iris {
			if iri.GetLink().Equals(it.GetLink(), false) {
				iris = append(iris[:k], iris[k+1:]...)
				break
			}
		}
		return iris, nil
	})
}

func addCollectionOnObject(r *repo, col pub.IRI) error {
	allStorageCollections := append(handlers.ActivityPubCollections, ap.FedboxCollections...)
	if ob, t := allStorageCollections.Split(col); handlers.ValidCollection(t) {
		// Create the collection on the object, if it doesn't exist
		if i, _ := r.LoadOne(ob); i != nil {
			if _, ok := t.AddTo(i); ok {
				_, err := r.SaveObject(i)
				return err
			}
		}
	}
	return nil
}

// AddToCollection
func (r *repo) AddToCollection(col pub.IRI, it pub.Item) error {
	addCollectionOnObject(r, col)
	return onCollection(r, col, it, func(iris pub.IRIs) (pub.IRIs, error) {
		if iris.Contains(it.GetLink()) {
			return iris, nil
		}
		return append(iris, it.GetLink()), nil
	})
}

// UpdateActor
func (r *repo) UpdateActor(it pub.Item) (pub.Item, error) {
	return r.UpdateObject(it)
}

// UpdateObject
func (r *repo) UpdateObject(it pub.Item) (pub.Item, error) {
	return r.SaveObject(it)
}

func (r *repo) DeleteActor(it pub.Item) (pub.Item, error) {
	return r.DeleteObject(it)
}

// DeleteObject
func (r *repo) DeleteObject(it pub.Item) (pub.Item, error) {
	var err error
	err = r.Open()
	if err != nil {
		return it, err
	}
	defer r.Close()
	var bucket handlers.CollectionType
	if pub.ActivityTypes.Contains(it.GetType()) {
		bucket = pathActivities
	} else if pub.ActorTypes.Contains(it.GetType()) {
		bucket = pathActors
	} else {
		bucket = pathObjects
	}
	if it, err = delete(r, it); err == nil {
		r.logFn(nil, "Added new %s: %s", bucket[:len(bucket)-1], it.GetLink())
	}
	return it, err
}

// GenerateID
func (r *repo) GenerateID(it pub.Item, by pub.Item) (pub.ID, error) {
	typ := it.GetType()

	var partOf string
	if pub.ActivityTypes.Contains(typ) {
		partOf = fmt.Sprintf("%s/%s", r.baseURL, ap.ActivitiesType)
	} else if pub.ActorTypes.Contains(typ) || typ == pub.ActorType {
		partOf = fmt.Sprintf("%s/%s", r.baseURL, ap.ActorsType)
	} else if pub.ObjectTypes.Contains(typ) {
		partOf = fmt.Sprintf("%s/%s", r.baseURL, ap.ObjectsType)
	}
	return ap.GenerateID(it, partOf, by)
}

func getMetadataKey(p []byte) []byte {
	return bytes.Join([][]byte{p, []byte(metaDataKey)}, sep)
}

// PasswordSet
func (r *repo) PasswordSet(it pub.Item, pw []byte) error {
	path := itemPath(it.GetLink())
	err := r.Open()
	if err != nil {
		return err
	}
	defer r.Close()

	err = r.d.Update(func(tx *badger.Txn) error {
		pw, err = bcrypt.GenerateFromPassword(pw, -1)
		if err != nil {
			return errors.Annotatef(err, "Could not encrypt the pw")
		}
		m := storage.Metadata{
			Pw: pw,
		}
		entryBytes, err := jsonld.Marshal(m)
		if err != nil {
			return errors.Annotatef(err, "Could not marshal metadata")
		}
		err = tx.Set(getMetadataKey(path), entryBytes)
		if err != nil {
			return errors.Annotatef(err, "Could not insert entry: %s", path)
		}
		return nil
	})

	return err
}

// PasswordCheck
func (r *repo) PasswordCheck(it pub.Item, pw []byte) error {
	path := itemPath(it.GetLink())
	err := r.Open()
	if err != nil {
		return err
	}
	defer r.Close()

	m := storage.Metadata{}
	err = r.d.View(func(tx *badger.Txn) error {
		i, err := tx.Get(getMetadataKey(path))
		if err != nil {
			return errors.Annotatef(err, "Could not find metadata in path %s", path)
		}
		i.Value(func(raw []byte) error {
			err := jsonld.Unmarshal(raw, &m)
			if err != nil {
				return errors.Annotatef(err, "Could not unmarshal metadata")
			}
			return nil
		})
		if err := bcrypt.CompareHashAndPassword(m.Pw, pw); err != nil {
			return errors.NewUnauthorized(err, "Invalid pw")
		}
		return nil
	})
	return err
}

// LoadMetadata
func (r *repo) LoadMetadata(iri pub.IRI) (*storage.Metadata, error) {
	err := r.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	path := itemPath(iri)

	var m *storage.Metadata
	err = r.d.View(func(tx *badger.Txn) error {
		i, err := tx.Get(getMetadataKey(path))
		if err != nil {
			return errors.Annotatef(err, "Could not find metadata in path %s", path)
		}
		m = new(storage.Metadata)
		return i.Value(func(raw []byte) error {
			return json.Unmarshal(raw, m)
		})
	})
	return m, err
}

// SaveMetadata
func (r *repo) SaveMetadata(m storage.Metadata, iri pub.IRI) error {
	err := r.Open()
	if err != nil {
		return err
	}
	defer r.Close()

	path := itemPath(iri)
	err = r.d.Update(func(tx *badger.Txn) error {
		entryBytes, err := jsonld.Marshal(m)
		if err != nil {
			return errors.Annotatef(err, "Could not marshal metadata")
		}
		err = tx.Set(getMetadataKey(path), entryBytes)
		if err != nil {
			return errors.Annotatef(err, "Could not insert entry: %s", path)
		}
		return nil
	})

	return err
}

const objectKey = "__raw"
const metaDataKey = "__meta_data"

func delete(r *repo, it pub.Item) (pub.Item, error) {
	if it.IsCollection() {
		err := pub.OnCollectionIntf(it, func(c pub.CollectionInterface) error {
			var err error
			for _, it := range c.Collection() {
				if it, err = delete(r, it); err != nil {
					return err
				}
			}
			return nil
		})
		return it, err
	}
	f := ap.FiltersNew()
	f.IRI = it.GetLink()
	if it.IsObject() {
		f.Type = []pub.ActivityVocabularyType{it.GetType()}
	}
	old, _ := r.loadOneFromPath(f)

	deleteCollections(r, old)
	t := pub.Tombstone{
		ID:   it.GetLink(),
		Type: pub.TombstoneType,
		To: pub.ItemCollection{
			pub.PublicNS,
		},
		Deleted:    time.Now().UTC(),
		FormerType: old.GetType(),
	}
	return save(r, t)
}

// createCollections
func createCollections(tx *badger.Txn, it pub.Item) error {
	if pub.ActorTypes.Contains(it.GetType()) {
		return pub.OnActor(it, func(p *pub.Actor) error {
			if p.Inbox != nil {
				p.Inbox, _ = createCollectionInPath(tx, p.Inbox)
			}
			if p.Outbox != nil {
				p.Outbox, _ = createCollectionInPath(tx, p.Outbox)
			}
			if p.Followers != nil {
				p.Followers, _ = createCollectionInPath(tx, p.Followers)
			}
			if p.Following != nil {
				p.Following, _ = createCollectionInPath(tx, p.Following)
			}
			if p.Liked != nil {
				p.Liked, _ = createCollectionInPath(tx, p.Liked)
			}
			return nil
		})
	}
	if pub.ObjectTypes.Contains(it.GetType()) {
		return pub.OnObject(it, func(o *pub.Object) error {
			if o.Replies != nil {
				o.Replies, _ = createCollectionInPath(tx, o.Replies)
			}
			if o.Likes != nil {
				o.Likes, _ = createCollectionInPath(tx, o.Likes)
			}
			if o.Shares != nil {
				o.Shares, _ = createCollectionInPath(tx, o.Shares)
			}
			return nil
		})
	}
	return nil
}

// deleteCollections
func deleteCollections(r *repo, it pub.Item) error {
	return r.d.Update(func(tx *badger.Txn) error {
		if pub.ActorTypes.Contains(it.GetType()) {
			return pub.OnActor(it, func(p *pub.Actor) error {
				var err error
				err = deleteCollectionFromPath(tx, handlers.Inbox.IRI(p))
				err = deleteCollectionFromPath(tx, handlers.Outbox.IRI(p))
				err = deleteCollectionFromPath(tx, handlers.Followers.IRI(p))
				err = deleteCollectionFromPath(tx, handlers.Following.IRI(p))
				err = deleteCollectionFromPath(tx, handlers.Liked.IRI(p))
				return err
			})
		}
		if pub.ObjectTypes.Contains(it.GetType()) {
			return pub.OnObject(it, func(o *pub.Object) error {
				var err error
				err = deleteCollectionFromPath(tx, handlers.Replies.IRI(o))
				err = deleteCollectionFromPath(tx, handlers.Likes.IRI(o))
				err = deleteCollectionFromPath(tx, handlers.Shares.IRI(o))
				return err
			})
		}
		return nil
	})
}

func save(r *repo, it pub.Item) (pub.Item, error) {
	itPath := itemPath(it.GetLink())
	err := r.d.Update(func(tx *badger.Txn) error {
		if err := createCollections(tx, it); err != nil {
			return errors.Annotatef(err, "could not create object's collections")
		}
		// TODO(marius): it's possible to set the encoding/decoding functions on the package or storage object level
		//  instead of using jsonld.(Un)Marshal like this.
		entryBytes, err := jsonld.Marshal(it)
		if err != nil {
			return errors.Annotatef(err, "could not marshal object")
		}
		k := getObjectKey(itPath)
		err = tx.Set(k, entryBytes)
		if err != nil {
			return errors.Annotatef(err, "could not store encoded object")
		}

		return nil
	})

	return it, err
}

func getCollectionKey(it pub.Item, h handlers.CollectionType) []byte {
	p := itemPath(it.GetLink())
	return bytes.Join([][]byte{p, []byte(h)}, sep)
}

var emptyCollection = []byte{'[', ']'}

func createCollectionInPath(b *badger.Txn, it pub.Item) (pub.Item, error) {
	if it == nil {
		return nil, nil
	}
	p := getObjectKey(itemPath(it.GetLink()))
	err := b.Set(p, emptyCollection)
	if err != nil {
		return nil, err
	}
	return it.GetLink(), nil
}

func deleteCollectionFromPath(b *badger.Txn, it pub.Item) error {
	if it == nil {
		return nil
	}
	p := getObjectKey(itemPath(it.GetLink()))
	return b.Delete(p)
}

func (r *repo) loadFromIterator(col *pub.ItemCollection, f s.Filterable) func(val []byte) error {
	isColFn := func(ff s.Filterable) bool {
		_, ok := ff.(pub.IRI)
		return ok
	}
	return func(val []byte) error {
		it, err := loadItem(val)
		if err != nil || it == nil {
			return errors.NewNotFound(err, "not found")
		}
		if !it.IsObject() && it.IsLink() {
			*col, err = r.loadItemsElements(f, it.GetLink())
			return err
		} else if it.IsCollection() {
			return pub.OnCollectionIntf(it, func(c pub.CollectionInterface) error {
				if isColFn(f) {
					f = c.Collection()
				}
				*col, err = r.loadItemsElements(f, c.Collection()...)
				return err
			})
		} else {
			if it.GetType() == pub.CreateType {
				// TODO(marius): this seems terribly not nice
				pub.OnActivity(it, func(a *pub.Activity) error {
					if !a.Object.IsObject() {
						ob, _ := r.loadOneFromPath(a.Object.GetLink())
						a.Object = ob
					}
					return nil
				})
			}
			it, err = ap.FilterIt(it, f)
			if err != nil {
				return err
			}
			if it != nil {
				*col = append(*col, it)
			}
		}
		return nil
	}
}

var sep = []byte{'/'}

func isObjectKey(k []byte) bool {
	return bytes.HasSuffix(k, []byte(objectKey))
}

func isMetadataKey(k []byte) bool {
	return bytes.HasSuffix(k, []byte(metaDataKey))
}

func isStorageCollectionKey(p []byte) bool {
	lst := handlers.CollectionType(path.Base(string(p)))
	return ap.FedboxCollections.Contains(lst)
}

func isIRIsKey(p []byte) bool {
	base := handlers.CollectionType(path.Base(string(p)))
	return ap.ValidCollection(base) && !ap.FedboxCollections.Contains(base)
}

func isItemKey(p []byte) bool {
	dir, file := path.Split(string(p))
	base := handlers.CollectionType(path.Base(dir))
	if base == "." {
		// special case for root path
		return true
	}
	u := uuid.Parse(file)
	return ap.ValidCollection(base) && !ap.FedboxCollections.Contains(base) && len(u) == 36
}

func iterKeyIsTooDeep(base, k []byte, depth int) bool {
	res := bytes.TrimPrefix(k, append(base, sep...))
	res = bytes.TrimSuffix(res, []byte(objectKey))
	cnt := bytes.Count(res, sep)
	return cnt > depth
}

func (r *repo) loadFromPath(f s.Filterable) (pub.ItemCollection, uint, error) {
	col := make(pub.ItemCollection, 0)
	err := r.d.View(func(tx *badger.Txn) error {
		iri := f.GetLink()
		fullPath := itemPath(iri)
		it := tx.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		depth := 0
		if isStorageCollectionKey(fullPath) {
			depth = 1
		}
		if handlers.ValidCollectionIRI(pub.IRI(fullPath)) {
			depth = 2
		}
		pathExists := false
		for it.Seek(fullPath); it.ValidForPrefix(fullPath); it.Next() {
			i := it.Item()
			k := i.Key()
			pathExists = true
			if iterKeyIsTooDeep(fullPath, k, depth) {
				continue
			}
			if isObjectKey(k) {
				err := i.Value(r.loadFromIterator(&col, f))
				if err != nil {
					continue
				}
			}
		}
		if !pathExists {
			return errors.NotFoundf("%s does not exist", fullPath)
		}
		return nil
	})

	return col, uint(len(col)), err
}

func (r *repo) LoadOne(f s.Filterable) (pub.Item, error) {
	err := r.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return r.loadOneFromPath(f)
}

func (r *repo) loadOneFromPath(f s.Filterable) (pub.Item, error) {
	col, cnt, err := r.loadFromPath(f)
	if err != nil {
		return nil, err
	}
	if cnt == 0 {
		return nil, errors.NotFoundf("nothing found")
	}
	return col.First(), nil
}

func getObjectKey(p []byte) []byte {
	return bytes.Join([][]byte{p, []byte(objectKey)}, sep)
}

func (r *repo) loadItemsElements(f s.Filterable, iris ...pub.Item) (pub.ItemCollection, error) {
	col := make(pub.ItemCollection, 0)
	err := r.d.View(func(tx *badger.Txn) error {
		for _, iri := range iris {
			it, err := r.loadItem(tx, itemPath(iri.GetLink()), f)
			if err != nil || it == nil {
				continue
			}
			col = append(col, it)
		}
		return nil
	})
	return col, err
}

func (r *repo) loadItem(b *badger.Txn, path []byte, f s.Filterable) (pub.Item, error) {
	i, err := b.Get(getObjectKey(path))
	if err != nil {
		return nil, errors.NewNotFound(err, "Unable to load path %s", path)
	}
	var raw []byte
	i.Value(func(val []byte) error {
		raw = val
		return nil
	})
	if raw == nil {
		return nil, nil
	}
	var it pub.Item
	it, err = loadItem(raw)
	if err != nil {
		return nil, err
	}
	if it == nil {
		return nil, errors.NotFoundf("not found")
	}
	if it.IsCollection() {
		// we need to dereference them, so no further filtering/processing is needed here
		return it, nil
	}
	if !it.IsObject() {
		it, _ = r.loadOneFromPath(it.GetLink())
	}
	if pub.ActivityTypes.Contains(it.GetType()) {
		pub.OnActivity(it, func(a *pub.Activity) error {
			if it.GetType() == pub.CreateType || ap.FiltersOnActivityObject(f) {
				// TODO(marius): this seems terribly not nice
				if a.Object != nil && !a.Object.IsObject() {
					a.Object, _ = r.loadOneFromPath(a.Object.GetLink())
				}
			}
			if ap.FiltersOnActivityActor(f) {
				// TODO(marius): this seems terribly not nice
				if a.Actor != nil && !a.Actor.IsObject() {
					a.Actor, _ = r.loadOneFromPath(a.Actor.GetLink())
				}
			}
			return nil
		})
	}
	if f != nil {
		return ap.FilterIt(it, f)
	}
	return it, nil
}

func loadItem(raw []byte) (pub.Item, error) {
	if raw == nil || len(raw) == 0 {
		// TODO(marius): log this instead of stopping the iteration and returning an error
		return nil, errors.Errorf("empty raw item")
	}
	return pub.UnmarshalJSON(raw)
}

func itemPath(iri pub.IRI) []byte {
	url, err := iri.URL()
	if err != nil {
		return nil
	}
	return []byte(path.Join(url.Host, url.Path))
}
