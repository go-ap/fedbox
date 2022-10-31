//go:build storage_badger || storage_all || (!storage_pgx && !storage_boltdb && !storage_fs && !storage_sqlite)

package badger

import (
	"bytes"
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"git.sr.ht/~mariusor/lw"
	"os"
	"path"

	"github.com/dgraph-io/badger/v3"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/cache"
	"github.com/go-ap/fedbox/storage"
	"github.com/go-ap/jsonld"
	"github.com/go-ap/processing"
	"golang.org/x/crypto/bcrypt"
)

const (
	pathActors     = ap.ActorsType
	pathActivities = ap.ActivitiesType
	pathObjects    = ap.ObjectsType
)

type repo struct {
	d       *badger.DB
	baseURL string
	path    string
	cache   cache.CanStore
	logger  lw.Logger
}

type loggerFn func(lw.Ctx, string, ...interface{})

// Config
type Config struct {
	Path    string
	BaseURL string
	Logger  lw.Logger
}

var emptyLogFn = func(lw.Ctx, string, ...interface{}) {}

// New returns a new repo repository
func New(c Config) (*repo, error) {
	var err error
	c.Path, err = Path(c)
	if err != nil {
		return nil, err
	}
	b := repo{
		path:    c.Path,
		baseURL: c.BaseURL,
		logger:  c.Logger,
	}
	return &b, nil
}

// Open opens the badger database if possible.
func (r *repo) Open() error {
	c := badger.DefaultOptions(r.path)
	if r.logger != nil {
		c = c.WithLogger(&logger{r.logger})
	}
	if r.path == "" {
		c.InMemory = true
	}
	c.MetricsEnabled = false

	var err error
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
	return r.d.Close()
}

// Load
func (r *repo) Load(i vocab.IRI) (vocab.Item, error) {
	var err error
	if r.Open(); err != nil {
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

	ret, err := r.loadFromPath(f, f.IsItemIRI())
	if len(ret) == 1 && f.IsItemIRI() {
		return ret.First(), err
	}
	return ret, err
}

func (r *repo) Create(col vocab.CollectionInterface) (vocab.CollectionInterface, error) {
	var err error
	err = r.Open()
	if err != nil {
		return col, err
	}
	defer r.Close()

	err = r.d.Update(func(tx *badger.Txn) error {
		_, err := createCollectionInPath(tx, col.GetLink())
		return err
	})
	return col, err
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
		r.logger.Infof("%s %s: %s", op, it.GetType(), it.GetLink())
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
		iris := make(vocab.IRIs, 0)

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

func addCollectionOnObject(r *repo, col vocab.IRI) error {
	allStorageCollections := append(vocab.ActivityPubCollections, ap.FedBOXCollections...)
	if ob, t := allStorageCollections.Split(col); vocab.ValidCollection(t) {
		// Create the collection on the object, if it doesn't exist
		if i, _ := r.LoadOne(ob); i != nil {
			if _, ok := t.AddTo(i); ok {
				_, err := r.Save(i)
				return err
			}
		}
	}
	return nil
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
	var err error
	err = r.Open()
	if err != nil {
		return err
	}
	defer r.Close()
	return delete(r, it)
}

func getMetadataKey(p []byte) []byte {
	return bytes.Join([][]byte{p, []byte(metaDataKey)}, sep)
}

// PasswordSet
func (r *repo) PasswordSet(it vocab.Item, pw []byte) error {
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
func (r *repo) PasswordCheck(it vocab.Item, pw []byte) error {
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
func (r *repo) LoadMetadata(iri vocab.IRI) (*storage.Metadata, error) {
	err := r.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	path := itemPath(iri)

	m := storage.Metadata{}
	err = r.d.View(func(tx *badger.Txn) error {
		i, err := tx.Get(getMetadataKey(path))
		if err != nil {
			return errors.Annotatef(err, "Could not find metadata in path %s", path)
		}
		return i.Value(func(raw []byte) error {
			return decodeFn(raw, &m)
		})
	})
	return &m, err
}

// SaveMetadata
func (r *repo) SaveMetadata(m storage.Metadata, iri vocab.IRI) error {
	err := r.Open()
	if err != nil {
		return err
	}
	defer r.Close()

	path := itemPath(iri)
	err = r.d.Update(func(tx *badger.Txn) error {
		entryBytes, err := encodeFn(m)
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

const objectKey = "__raw"
const metaDataKey = "__meta_data"

func delete(r *repo, it vocab.Item) error {
	if it.IsCollection() {
		return vocab.OnCollectionIntf(it, func(c vocab.CollectionInterface) error {
			for _, it := range c.Collection() {
				if err := delete(r, it); err != nil {
					r.logger.Infof("Unable to remove item %s", it.GetLink())
				}
			}
			return nil
		})
	}
	f := ap.FiltersNew()
	f.IRI = it.GetLink()
	if it.IsObject() {
		f.Type = ap.CompStrs{ap.StringEquals(string(it.GetType()))}
	}
	old, err := r.loadOneFromPath(f)
	if err != nil {
		return err
	}

	return r.d.Update(func(tx *badger.Txn) error {
		return deleteFromPath(r, tx, old)
	})
}

// createCollections
func createCollections(tx *badger.Txn, it vocab.Item) error {
	if vocab.IsNil(it) || !it.IsObject() {
		return nil
	}
	if vocab.ActorTypes.Contains(it.GetType()) {
		vocab.OnActor(it, func(p *vocab.Actor) error {
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
	return vocab.OnObject(it, func(o *vocab.Object) error {
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

// deleteCollections
func deleteCollections(r *repo, it vocab.Item) error {
	return r.d.Update(func(tx *badger.Txn) error {
		if vocab.ActorTypes.Contains(it.GetType()) {
			return vocab.OnActor(it, func(p *vocab.Actor) error {
				var err error
				err = deleteFromPath(r, tx, vocab.Inbox.IRI(p))
				err = deleteFromPath(r, tx, vocab.Outbox.IRI(p))
				err = deleteFromPath(r, tx, vocab.Followers.IRI(p))
				err = deleteFromPath(r, tx, vocab.Following.IRI(p))
				err = deleteFromPath(r, tx, vocab.Liked.IRI(p))
				return err
			})
		}
		if vocab.ObjectTypes.Contains(it.GetType()) {
			return vocab.OnObject(it, func(o *vocab.Object) error {
				var err error
				err = deleteFromPath(r, tx, vocab.Replies.IRI(o))
				err = deleteFromPath(r, tx, vocab.Likes.IRI(o))
				err = deleteFromPath(r, tx, vocab.Shares.IRI(o))
				return err
			})
		}
		return nil
	})
}

func save(r *repo, it vocab.Item) (vocab.Item, error) {
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

var emptyCollection = []byte{'[', ']'}

func createCollectionInPath(b *badger.Txn, it vocab.Item) (vocab.Item, error) {
	if vocab.IsNil(it) {
		return nil, nil
	}
	p := getObjectKey(itemPath(it.GetLink()))
	err := b.Set(p, emptyCollection)
	if err != nil {
		return nil, err
	}
	return it.GetLink(), nil
}

func deleteFromPath(r *repo, b *badger.Txn, it vocab.Item) error {
	if vocab.IsNil(it) {
		return nil
	}
	p := getObjectKey(itemPath(it.GetLink()))
	if err := b.Delete(p); err != nil {
		return err
	}
	r.cache.Remove(it.GetLink())
	return nil
}

func (r *repo) loadFromIterator(col *vocab.ItemCollection, f processing.Filterable) func(val []byte) error {
	isColFn := func(ff processing.Filterable) bool {
		_, ok := ff.(vocab.IRI)
		return ok
	}
	return func(val []byte) error {
		it, err := loadItem(val)
		if err != nil || vocab.IsNil(it) {
			return errors.NewNotFound(err, "not found")
		}
		if !it.IsObject() && it.IsLink() {
			c, err := r.loadItemsElements(f, it.GetLink())
			if err != nil {
				return err
			}
			for _, it := range c {
				if col.Contains(it.GetLink()) {
					continue
				}
				*col = append(*col, it)
			}
		} else if it.IsCollection() {
			return vocab.OnCollectionIntf(it, func(ci vocab.CollectionInterface) error {
				if isColFn(f) {
					f = ci.Collection()
				}
				c, err := r.loadItemsElements(f, ci.Collection()...)
				if err != nil {
					return err
				}
				for _, it := range c {
					if col.Contains(it.GetLink()) {
						continue
					}
					*col = append(*col, it)
				}
				return nil
			})
		} else {
			if it.GetType() == vocab.CreateType {
				// TODO(marius): this seems terribly not nice
				vocab.OnActivity(it, func(a *vocab.Activity) error {
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
				if vocab.ActorTypes.Contains(it.GetType()) {
					vocab.OnActor(it, loadFilteredPropsForActor(r, f))
				}
				if vocab.ObjectTypes.Contains(it.GetType()) {
					vocab.OnObject(it, loadFilteredPropsForObject(r, f))
				}
				if vocab.IntransitiveActivityTypes.Contains(it.GetType()) {
					vocab.OnIntransitiveActivity(it, loadFilteredPropsForIntransitiveActivity(r, f))
				}
				if vocab.ActivityTypes.Contains(it.GetType()) {
					vocab.OnActivity(it, loadFilteredPropsForActivity(r, f))
				}
				if !col.Contains(it.GetLink()) {
					*col = append(*col, it)
				}
			}
		}
		return nil
	}
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
				if ob, err := r.loadOneFromPath(t.GetLink()); err == nil {
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
			if ob, err := r.loadOneFromPath(a.Object.GetLink()); err == nil {
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
			if act, err := r.loadOneFromPath(a.Actor.GetLink()); err == nil {
				if act, _ = ap.FilterIt(act, fa); act != nil {
					a.Actor = act
				}
			}
		}
		if ok, ft := ap.FiltersOnActivityTarget(f); ok && !vocab.IsNil(a.Target) && vocab.IsIRI(a.Target) {
			if t, err := r.loadOneFromPath(a.Target.GetLink()); err == nil {
				if t, _ = ap.FilterIt(t, ft); t != nil {
					a.Target = t
				}
			}
		}
		return nil
	}
}

var sep = []byte{'/'}

func isObjectKey(k []byte) bool {
	return bytes.HasSuffix(k, []byte(objectKey))
}

func isStorageCollectionKey(p []byte) bool {
	lst := vocab.CollectionPath(path.Base(string(p)))
	return vocab.CollectionPaths{ap.ActivitiesType, ap.ActorsType, ap.ObjectsType}.Contains(lst)
}

func iterKeyIsTooDeep(base, k []byte, depth int) bool {
	res := bytes.TrimPrefix(k, append(base, sep...))
	res = bytes.TrimSuffix(res, []byte(objectKey))
	cnt := bytes.Count(res, sep)
	return cnt > depth
}

func (r *repo) loadFromPath(f processing.Filterable, loadMaxOne bool) (vocab.ItemCollection, error) {
	col := make(vocab.ItemCollection, 0)

	err := r.d.View(func(tx *badger.Txn) error {
		iri := f.GetLink()
		fullPath := itemPath(iri)

		depth := 0
		if isStorageCollectionKey(fullPath) {
			depth = 1
		}
		if vocab.ValidCollectionIRI(vocab.IRI(fullPath)) {
			depth = 2
		}
		opt := badger.DefaultIteratorOptions
		opt.Prefix = fullPath
		it := tx.NewIterator(opt)
		defer it.Close()
		pathExists := false
		for it.Seek(fullPath); it.ValidForPrefix(fullPath); it.Next() {
			i := it.Item()
			k := i.Key()
			pathExists = true
			if iterKeyIsTooDeep(fullPath, k, depth) {
				continue
			}
			if isObjectKey(k) {
				if err := i.Value(r.loadFromIterator(&col, f)); err != nil {
					r.logger.WithContext(lw.Ctx{"k": k, "err": err.Error()}).Errorf("unable to load")
					continue
				}
				if len(col) == 1 && loadMaxOne {
					break
				}
			}
		}
		if !pathExists && len(col) == 0 {
			return errors.NotFoundf("%s does not exist", fullPath)
		}
		return nil
	})

	return col, err
}

func (r *repo) LoadOne(f processing.Filterable) (vocab.Item, error) {
	err := r.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return r.loadOneFromPath(f)
}

func (r *repo) loadOneFromPath(f processing.Filterable) (vocab.Item, error) {
	col, err := r.loadFromPath(f, true)
	if err != nil {
		return nil, err
	}
	if len(col) == 0 {
		return nil, errors.NotFoundf("nothing found")
	}
	return col.First(), nil
}

func getObjectKey(p []byte) []byte {
	return bytes.Join([][]byte{p, []byte(objectKey)}, sep)
}

func (r *repo) loadItemsElements(f processing.Filterable, iris ...vocab.Item) (vocab.ItemCollection, error) {
	col := make(vocab.ItemCollection, 0)
	err := r.d.View(func(tx *badger.Txn) error {
		for _, iri := range iris {
			it, err := r.loadItem(tx, itemPath(iri.GetLink()), f)
			if err != nil || vocab.IsNil(it) || col.Contains(it.GetLink()) {
				continue
			}
			col = append(col, it)
		}
		return nil
	})
	return col, err
}

func (r *repo) loadItem(b *badger.Txn, path []byte, f processing.Filterable) (vocab.Item, error) {
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
	var it vocab.Item
	it, err = loadItem(raw)
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
		it, _ = r.loadOneFromPath(it.GetLink())
	}
	if f != nil {
		return ap.FilterIt(it, f)
	}
	return it, nil
}

func loadItem(raw []byte) (vocab.Item, error) {
	if raw == nil || len(raw) == 0 {
		// TODO(marius): log this instead of stopping the iteration and returning an error
		return nil, errors.Errorf("empty raw item")
	}
	return vocab.UnmarshalJSON(raw)
}

func itemPath(iri vocab.IRI) []byte {
	url, err := iri.URL()
	if err != nil {
		return nil
	}
	return []byte(path.Join(url.Host, url.Path))
}

func (r *repo) CreateService(service vocab.Service) error {
	err := r.Open()
	defer r.Close()
	if err != nil {
		return err
	}
	if it, err := save(r, service); err == nil {
		op := "Updated"
		id := it.GetID()
		if !id.IsValid() {
			op = "Added new"
		}
		r.logger.Infof("%s %s: %s", op, it.GetType(), it.GetLink())
	}
	return err
}

func Path(c Config) (string, error) {
	if c.Path == "" {
		return "", nil
	}
	return c.Path, mkDirIfNotExists(c.Path)
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
