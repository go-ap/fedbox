package fs

import (
	"fmt"
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/storage"
	"github.com/go-ap/handlers"
	"github.com/go-ap/jsonld"
	s "github.com/go-ap/storage"
	"github.com/mariusor/qstring"
	"github.com/pborman/uuid"
	"golang.org/x/crypto/bcrypt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var errNotImplemented = errors.NotImplementedf("not implemented")

type loggerFn func(string, ...interface{})

var defaultLogFn = func(string, ...interface{}) {}

func getAbsStoragePath(p string) (string, error) {
	if !filepath.IsAbs(p) {
		var err error
		p, err = filepath.Abs(p)
		if err != nil {
			return "", err
		}
	} else {
		p = p
	}
	if fi, err := os.Stat(p); err != nil {
		return "", err
	} else if !fi.IsDir() {
		return "", errors.NotValidf("path %s is invalid for storage", p)
	}
	return p, nil
}

// New returns a new repo repository
func New(c config.Options) (*repo, error) {
	p, err := getAbsStoragePath(c.StoragePath)
	if err != nil {
		return nil, err
	}
	p = path.Clean(path.Join(p, string(c.Env)))
	if err := mkDirIfNotExists(p); err != nil {
		return nil, err
	}
	b := repo{
		m:       sync.Mutex{},
		path:    p,
		baseURL: c.BaseURL,
		logFn:   defaultLogFn,
		errFn:   defaultLogFn,
	}
	b.prevPath, _ = os.Getwd()
	return &b, nil
}

type repo struct {
	m        sync.Mutex
	baseURL  string
	path     string
	prevPath string
	logFn    loggerFn
	errFn    loggerFn
}

// Open
func (r *repo) Open() error {
	r.m.Lock()
	return os.Chdir(r.path)
}

// Close
func (r *repo) Close() error {
	defer r.m.Unlock()
	return os.Chdir(r.prevPath)
}

// LoadOne
func (r *repo) LoadOne(f s.Filterable) (pub.Item, error) {
	err := r.Open()
	defer r.Close()
	if err != nil {
		return nil, err
	}
	return loadOneFromPath(f)
}

func (r *repo) CreateService(service pub.Service) error {
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
		r.logFn("%s %s: %s", op, it.GetType(), it.GetLink())
	}
	return err
}

// Load
func (r *repo) Load(f s.Filterable) (pub.ItemCollection, uint, error) {
	err := r.Open()
	defer r.Close()
	if err != nil {
		return nil, 0, err
	}

	return loadFromPath(f)
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

// LoadCollection
func (r *repo) LoadCollection(f s.Filterable) (pub.CollectionInterface, error) {
	err := r.Open()
	defer r.Close()
	if err != nil {
		return nil, err
	}

	iri := f.GetLink()
	url, err := iri.URL()
	if err != nil {
		r.errFn("invalid IRI filter element %s when loading collections", iri)
	}

	q, _ := qstring.Marshal(f)
	url.RawQuery = q.Encode()

	col := &pub.OrderedCollection{}
	col.ID = pub.ID(url.String())
	col.Type = pub.OrderedCollectionType

	elements, count, err := loadFromPath(f)
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

// CreateCollection
func (r *repo) CreateCollection(col pub.CollectionInterface) (pub.CollectionInterface, error) {
	return nil, errNotImplemented
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
	err := r.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	if it, err = save(r, it); err == nil {
		op := "Updated"
		id := it.GetID()
		if !id.IsValid() {
			op = "Added new"
		}
		r.logFn("%s %s: %s", op, it.GetType(), it.GetLink())
	}
	return it, err
}

// RemoveFromCollection
func (r *repo) RemoveFromCollection(col pub.IRI, it pub.Item) error {
	err := r.Open()
	defer r.Close()
	if err != nil {
		return err
	}

	ob, t := path.Split(col.String())
	var link pub.IRI
	if ap.ValidCollection(handlers.CollectionType(t)) {
		ob = strings.TrimRight(ob, "/")
		// Create the collection on the object, if it doesn't exist
		i, err := loadOneFromPath(pub.IRI(ob))
		if err != nil {
			return err
		}
		if p, ok := handlers.CollectionType(t).AddTo(i); ok {
			save(r, i)
			link = p
		} else {
			link = handlers.CollectionType(t).IRI(i)
		}
	}

	linkPath := itemPath(link)
	name := path.Base(itemPath(it.GetLink()))
	// we create a symlink to the persisted object in the current collection
	return onCollection(r, col, it, func(p string) error {
		inCollection := false
		if fileInfo, err := ioutil.ReadDir(p); err == nil {
			for _, fi := range fileInfo {
				if fi.Name() == name && (fi.Mode()&os.ModeSymlink == os.ModeSymlink) {
					inCollection = true
				}
			}
		}
		if inCollection {
			link := path.Join(linkPath, name)
			return os.RemoveAll(link)
		}
		return nil
	})
}

// AddToCollection
func (r *repo) AddToCollection(col pub.IRI, it pub.Item) error {
	err := r.Open()
	defer r.Close()
	if err != nil {
		return err
	}

	ob, t := path.Split(col.String())
	var link pub.IRI
	if isStorageCollectionKey(t) {
		ob = strings.TrimRight(ob, "/")
		// Create the collection on the object, if it doesn't exist
		i, err := loadOneFromPath(pub.IRI(ob))
		if err != nil {
			return err
		}
		if p, ok := handlers.CollectionType(t).AddTo(i); ok {
			save(r, i)
			link = p
		} else {
			link = handlers.CollectionType(t).IRI(i)
		}
	} else {
		return errors.Newf("Invalid collection %s", t)
	}

	linkPath := itemPath(link)
	itPath := itemPath(it.GetLink())
	fullLink := path.Join(linkPath, path.Base(itPath))

	// we create a symlink to the persisted object in the current collection
	return onCollection(r, col, it, func(p string) error {
		err := mkDirIfNotExists(p)
		if err != nil {
			return errors.Annotatef(err, "Unable to create collection folder %s", p)
		}
		inCollection := false
		if fileInfo, err := ioutil.ReadDir(p); err == nil {
			for _, fi := range fileInfo {
				if fi.Name() == fullLink && (fi.Mode()&os.ModeSymlink == os.ModeSymlink) {
					inCollection = true
				}
			}
		}
		if inCollection {
			return nil
		}

		absItPath, err := filepath.Abs(itPath)
		if err != nil {
			return err
		}
		absLinkPath, err := filepath.Abs(fullLink)
		if err != nil {
			return err
		}
		return os.Symlink(absItPath, absLinkPath)
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
	err := r.Open()
	defer r.Close()
	if err != nil {
		return nil, err
	}

	f := ap.FiltersNew()
	f.IRI = it.GetLink()

	t := pub.Tombstone{
		ID:   it.GetLink(),
		Type: pub.TombstoneType,
		To: pub.ItemCollection{
			pub.PublicNS,
		},
		Deleted: time.Now().UTC(),
	}

	if it.IsObject() {
		t.FormerType = it.GetType()
	} else {
		if old, err := loadOneFromPath(f); err == nil {
			t.FormerType = old.GetType()
		}
	}

	deleteCollections(it)
	return save(r, t)
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

// PasswordSet
func (r *repo) PasswordSet(it pub.Item, pw []byte) error {
	pw, err := bcrypt.GenerateFromPassword(pw, -1)
	if err != nil {
		return errors.Annotatef(err, "could not generate pw hash")
	}
	m := storage.Metadata{
		Pw: pw,
	}
	return r.SaveMetadata(m, it.GetLink())
}

// PasswordCheck
func (r *repo) PasswordCheck(it pub.Item, pw []byte) error {
	m, err := r.LoadMetadata(it.GetLink())
	if err != nil {
		return errors.Annotatef(err, "Could not find load metadata for %s", it)
	}
	if err := bcrypt.CompareHashAndPassword(m.Pw, pw); err != nil {
		return errors.NewUnauthorized(err, "Invalid pw")
	}
	return err
}

// LoadMetadata
func (r *repo) LoadMetadata(iri pub.IRI) (*storage.Metadata, error) {
	err := r.Open()
	defer r.Close()
	if err != nil {
		return nil, err
	}

	m := new(storage.Metadata)
	p := itemPath(iri)
	raw, err := loadRawFromPath(getMetadataKey(p))
	if err != nil {
		return nil, errors.Annotatef(err, "Could not find metadata in path %s", p)
	}
	err = jsonld.Unmarshal(raw, m)
	if err != nil {
		return nil, errors.Annotatef(err, "Could not unmarshal metadata")
	}
	return m, nil
}

// SaveMetadata
func (r *repo) SaveMetadata(m storage.Metadata, iri pub.IRI) error {
	err := r.Open()
	defer r.Close()
	if err != nil {
		return err
	}

	p := getMetadataKey(itemPath(iri))
	f, err := createOrOpenFile(p)
	if err != nil {
		return err
	}
	defer f.Close()

	entryBytes, err := jsonld.Marshal(m)
	if err != nil {
		return errors.Annotatef(err, "Could not marshal metadata")
	}
	wrote, err := f.Write(entryBytes)
	if err != nil {
		return errors.Annotatef(err, "could not store encoded object")
	}
	if wrote != len(entryBytes) {
		return errors.Annotatef(err, "failed writing full object")
	}
	return nil
}

func createOrOpenFile(p string) (*os.File, error) {
	f, err := os.Open(p)
	if err != nil {
		if os.IsNotExist(err) {
			// create json file
			return os.Create(p)
		} else {
			return f, err
		}
	}
	return f, err
}

func isStorageCollectionKey(p string) bool {
	lst := handlers.CollectionType(path.Base(p))
	return ap.FedboxCollections.Contains(lst) || handlers.OnActor.Contains(lst) || handlers.OnObject.Contains(lst)
}

func isIDKey(p string) bool {
	dir, file := path.Split(p)
	if isStorageCollectionKey(dir) {
		if id := uuid.Parse(file).NodeID(); len(id) > 0 {
			return true
		}
	}
	return false
}

func itemPath(iri pub.IRI) string {
	url, err := iri.URL()
	if err != nil {
		return ""
	}
	//p := strings.ReplaceAll(url.Path, "-", "/")
	p := url.Path
	return path.Join(url.Host, p)
}

// createCollections
func createCollections(it pub.Item) error {
	if pub.ActorTypes.Contains(it.GetType()) {
		return pub.OnActor(it, func(p *pub.Actor) error {
			if p.Inbox != nil {
				p.Inbox, _ = createCollectionInPath(p.Inbox)
			}
			if p.Outbox != nil {
				p.Outbox, _ = createCollectionInPath(p.Outbox)
			}
			if p.Followers != nil {
				p.Followers, _ = createCollectionInPath(p.Followers)
			}
			if p.Following != nil {
				p.Following, _ = createCollectionInPath(p.Following)
			}
			if p.Liked != nil {
				p.Liked, _ = createCollectionInPath(p.Liked)
			}
			return nil
		})
	}
	if pub.ObjectTypes.Contains(it.GetType()) {
		return pub.OnObject(it, func(o *pub.Object) error {
			if o.Replies != nil {
				o.Replies, _ = createCollectionInPath(o.Replies)
			}
			if o.Likes != nil {
				o.Likes, _ = createCollectionInPath(o.Likes)
			}
			if o.Shares != nil {
				o.Shares, _ = createCollectionInPath(o.Shares)
			}
			return nil
		})
	}
	return nil
}

const (
	objectKey   = "__raw.json"
	metaDataKey = "__meta_data.json"
)

func isObjectKey(k string) bool {
	return strings.HasSuffix(k, objectKey)
}

func isMetadataKey(k string) bool {
	return strings.HasSuffix(k, metaDataKey)
}

func getMetadataKey(p string) string {
	return path.Join(p, metaDataKey)
}

func getObjectKey(p string) string {
	return path.Join(p, objectKey)
}

func createCollectionInPath(it pub.Item) (pub.Item, error) {
	itPath := itemPath(it.GetLink())
	return it.GetLink(), mkDirIfNotExists(itPath)
}

func deleteCollectionFromPath(it pub.Item) error {
	if it == nil {
		return nil
	}
	itPath := itemPath(it.GetLink())
	if fi, err := os.Stat(itPath); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else if fi.IsDir() {
		return os.Remove(itPath)
	}
	return nil
}

// deleteCollections
func deleteCollections(it pub.Item) error {
	if pub.ActorTypes.Contains(it.GetType()) {
		return pub.OnActor(it, func(p *pub.Actor) error {
			var err error
			err = deleteCollectionFromPath(handlers.Inbox.IRI(p))
			err = deleteCollectionFromPath(handlers.Outbox.IRI(p))
			err = deleteCollectionFromPath(handlers.Followers.IRI(p))
			err = deleteCollectionFromPath(handlers.Following.IRI(p))
			err = deleteCollectionFromPath(handlers.Liked.IRI(p))
			return err
		})
	}
	if pub.ObjectTypes.Contains(it.GetType()) {
		return pub.OnObject(it, func(o *pub.Object) error {
			var err error
			err = deleteCollectionFromPath(handlers.Replies.IRI(o))
			err = deleteCollectionFromPath(handlers.Likes.IRI(o))
			err = deleteCollectionFromPath(handlers.Shares.IRI(o))
			return err
		})
	}
	return nil
}

func mkDirIfNotExists(p string) error {
	if fi, err := os.Stat(p); err != nil {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(p, os.ModeDir|os.ModePerm|0700); err != nil {
				return err
			}
		}
	} else if !fi.IsDir() {
		return errors.Errorf("path exists, and is not a folder %s", p)
	}
	return nil
}

func save(r *repo, it pub.Item) (pub.Item, error) {
	itPath := itemPath(it.GetLink())
	mkDirIfNotExists(itPath)

	if err := createCollections(it); err != nil {
		return it, errors.Annotatef(err, "could not create object's collections")
	}
	// TODO(marius): it's possible to set the encoding/decoding functions on the package or storage object level
	//  instead of using jsonld.(Un)Marshal like this.
	entryBytes, err := jsonld.Marshal(it)
	if err != nil {
		return it, errors.Annotatef(err, "could not marshal object")
	}

	// create json file
	f, err := os.Create(getObjectKey(itPath))
	if err != nil {
		return it, errors.Annotatef(err, "could not create file")
	}
	defer f.Close()
	wrote, err := f.Write(entryBytes)
	if err != nil {
		return it, errors.Annotatef(err, "could not store encoded object")
	}
	if wrote != len(entryBytes) {
		return it, errors.Annotatef(err, "failed writing object")
	}

	return it, nil
}

func onCollection(r *repo, col pub.IRI, it pub.Item, fn func(p string) error) error {
	if it == nil {
		return errors.Newf("Unable to operate on nil element")
	}
	if len(col) == 0 {
		return errors.Newf("Unable to find collection")
	}
	if len(it.GetLink()) == 0 {
		return errors.Newf("Invalid collection, it does not have a valid IRI")
	}
	if !col.GetLink().Contains(pub.IRI(r.baseURL), false) {
		return errors.Newf("Unable to save to non local collection %s", col)
	}

	itPath := itemPath(col)
	err := fn(itPath)
	if err != nil {
		return errors.Annotatef(err, "Unable to save entries to collection %s", itPath)
	}
	return err
}

func loadRawFromPath(itPath string) ([]byte, error) {
	f, err := os.Open(itPath)
	if err != nil {
		return nil, errors.Annotatef(err, "Unable find path %s", itPath)
	}
	fi, err := f.Stat()
	if err != nil {
		return nil, errors.Annotatef(err, "Unable stat file at path %s", itPath)
	}
	raw := make([]byte, fi.Size())
	cnt, err := f.Read(raw)
	if err != nil {
		return nil, errors.Annotatef(err, "Unable read file at path %s", itPath)
	}
	if cnt != len(raw) {
		return nil, errors.Annotatef(err, "Unable read the whole file at path %s", itPath)
	}
	return raw, nil
}

func loadFromRaw(raw []byte) (pub.Item, error) {
	if raw == nil || len(raw) == 0 {
		// TODO(marius): log this instead of stopping the iteration and returning an error
		return nil, errors.Errorf("empty raw item")
	}
	return pub.UnmarshalJSON(raw)
}

func loadOneFromPath(f s.Filterable) (pub.Item, error) {
	col, cnt, err := loadFromPath(f)
	if err != nil {
		return nil, err
	}
	if cnt == 0 {
		return nil, errors.NotFoundf("nothing found")
	}
	return col.First(), nil
}

func loadItem(p string, f s.Filterable) (pub.Item, error) {
	raw, err := loadRawFromPath(p)
	if raw == nil {
		return nil, nil
	}
	var it pub.Item
	it, err = loadFromRaw(raw)
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
		it, _ = loadOneFromPath(it.GetLink())
	}
	if it.GetType() == pub.CreateType {
		// TODO(marius): this seems terribly not nice
		pub.OnActivity(it, func(a *pub.Activity) error {
			if a.Object == nil {
				return nil
			}
			if !a.Object.IsObject() {
				ob, _ := loadOneFromPath(a.Object.GetLink())
				a.Object = ob
			}
			return nil
		})
	}
	if f != nil {
		return ap.FilterIt(it, f)
	}
	return it, nil
}

func loadFromPath(f s.Filterable) (pub.ItemCollection, uint, error) {
	var err error
	col := make(pub.ItemCollection, 0)

	itPath := itemPath(f.GetLink())
	if isStorageCollectionKey(itPath) {
		err = filepath.Walk(itPath, func(p string, info os.FileInfo, err error) error {
			if err != nil && os.IsNotExist(err)  {
				return errors.NotFoundf("%s not found", p)
			}
			dirPath, _ := path.Split(p)
			dir := strings.TrimRight(dirPath, "/")
			if dir != itPath {
				return nil
			}
			if _, ok := f.(pub.IRI); ok {
				// when loading a collection by path, we want to avoid filtering out IRIs that don't specifically
				// contain the path, so we nil the filter
				f = nil
			}
			it, _ := loadItem(getObjectKey(p), f)
			if it != nil {
				col = append(col, it)
			}
			return nil
		})
	} else {
		var it pub.Item
		it, err = loadItem(getObjectKey(itPath), f)
		if err != nil {
			return nil, 0, errors.NewNotFound(err, "not found")
		}
		if it != nil {
			col = append(col, it)
		}
	}
	return col, uint(len(col)), err
}

func orderItems(col pub.ItemCollection) pub.ItemCollection {
	sort.SliceStable(col, func(i, j int) bool {
		return pub.ItemOrderTimestamp(col[i], col[j])
	})
	return col
}
