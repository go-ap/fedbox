// +build storage_fs storage_all !storage_boltdb,!storage_badger,!storage_pgx

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
	"time"
)

var encodeFn = jsonld.Marshal
var decodeFn = jsonld.Unmarshal

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
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	b := repo{
		path:    p,
		cwd:     cwd,
		baseURL: c.BaseURL,
		logFn:   defaultLogFn,
		errFn:   defaultLogFn,
	}
	return &b, nil
}

type repo struct {
	baseURL string
	path    string
	cwd     string
	logFn   loggerFn
	errFn   loggerFn
}

// Open
func (r *repo) Open() error {
	return os.Chdir(r.path)
}

// Close
func (r *repo) Close() error {
	return os.Chdir(r.cwd)
}

// LoadOne
func (r *repo) LoadOne(f s.Filterable) (pub.Item, error) {
	err := r.Open()
	defer r.Close()
	if err != nil {
		return nil, err
	}
	return r.loadOneFromPath(f)
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

// CreateCollection
func (r *repo) CreateCollection(col pub.CollectionInterface) (pub.CollectionInterface, error) {
	if col == nil {
		return col, errors.Newf("Unable to operate on nil element")
	}
	if len(col.GetLink()) == 0 {
		return col, errors.Newf("Invalid collection, it does not have a valid IRI")
	}
	return col, pub.OnCollection(col, func(c *pub.Collection) error {
		return mkDirIfNotExists(r.itemPath(c.GetLink()))
	})
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

	ob, t := handlers.Split(col)
	var link pub.IRI
	if ap.ValidCollection(t) {
		// Create the collection on the object, if it doesn't exist
		i, err := r.loadOneFromPath(ob)
		if err != nil {
			return err
		}
		if p, ok := t.AddTo(i); ok {
			save(r, i)
			link = p
		} else {
			link = t.IRI(i)
		}
	}

	linkPath := r.itemPath(link)
	name := path.Base(r.itemPath(it.GetLink()))
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

func addCollectionOnObject(r *repo, col pub.IRI) error {
	if ob, t := handlers.Split(col); handlers.ValidCollection(t) {
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
	err := r.Open()
	defer r.Close()
	if err != nil {
		return err
	}

	allStorageCollections := append(handlers.ActivityPubCollections, ap.FedboxCollections...)
	ob, t := allStorageCollections.Split(col)
	var link pub.IRI
	if isStorageCollectionKey(string(t)) {
		// Create the collection on the object, if it doesn't exist
		i, err := r.loadOneFromPath(ob)
		if err != nil {
			return err
		}
		if p, ok := t.AddTo(i); ok {
			save(r, i)
			link = p
		} else {
			link = t.IRI(i)
		}
	} else {
		return errors.Newf("Invalid collection %s", t)
	}

	linkPath := r.itemPath(link)
	itPath := r.itemPath(it.GetLink())
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

	if it.IsCollection() {
		err := pub.OnCollectionIntf(it, func(c pub.CollectionInterface) error {
			var err error
			for _, it := range c.Collection() {
				if it, err = r.DeleteObject(it); err != nil {
					return err
				}
			}
			return nil
		})
		return it, err
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
		if old, err := r.loadOneFromPath(f); err == nil {
			t.FormerType = old.GetType()
		}
	}

	deleteCollections(*r, it)
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
	p := r.itemPath(iri)
	raw, err := loadRawFromPath(getMetadataKey(p))
	if err != nil {
		return nil, errors.Annotatef(err, "Could not find metadata in path %s", p)
	}
	err = decodeFn(raw, m)
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

	p := getMetadataKey(r.itemPath(iri))
	f, err := createOrOpenFile(p)
	if err != nil {
		return err
	}
	defer f.Close()

	entryBytes, err := encodeFn(m)
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
	return os.OpenFile(p, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
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

func (r repo) itemPath(iri pub.IRI) string {
	url, err := iri.URL()
	if err != nil {
		return ""
	}
	p := url.Path
	return path.Join(r.path, url.Host, p)
}

// createCollections
func createCollections(r repo, it pub.Item) error {
	if pub.ActorTypes.Contains(it.GetType()) {
		return pub.OnActor(it, func(p *pub.Actor) error {
			if p.Inbox != nil {
				p.Inbox, _ = createCollectionInPath(r, p.Inbox)
			}
			if p.Outbox != nil {
				p.Outbox, _ = createCollectionInPath(r, p.Outbox)
			}
			if p.Followers != nil {
				p.Followers, _ = createCollectionInPath(r, p.Followers)
			}
			if p.Following != nil {
				p.Following, _ = createCollectionInPath(r, p.Following)
			}
			if p.Liked != nil {
				p.Liked, _ = createCollectionInPath(r, p.Liked)
			}
			return nil
		})
	}
	if pub.ObjectTypes.Contains(it.GetType()) {
		return pub.OnObject(it, func(o *pub.Object) error {
			if o.Replies != nil {
				o.Replies, _ = createCollectionInPath(r, o.Replies)
			}
			if o.Likes != nil {
				o.Likes, _ = createCollectionInPath(r, o.Likes)
			}
			if o.Shares != nil {
				o.Shares, _ = createCollectionInPath(r, o.Shares)
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

func createCollectionInPath(r repo, it pub.Item) (pub.Item, error) {
	itPath := r.itemPath(it.GetLink())
	return it.GetLink(), mkDirIfNotExists(itPath)
}

func deleteCollectionFromPath(r repo, it pub.Item) error {
	if it == nil {
		return nil
	}
	itPath := r.itemPath(it.GetLink())
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
func deleteCollections(r repo, it pub.Item) error {
	if pub.ActorTypes.Contains(it.GetType()) {
		return pub.OnActor(it, func(p *pub.Actor) error {
			var err error
			err = deleteCollectionFromPath(r, handlers.Inbox.IRI(p))
			err = deleteCollectionFromPath(r, handlers.Outbox.IRI(p))
			err = deleteCollectionFromPath(r, handlers.Followers.IRI(p))
			err = deleteCollectionFromPath(r, handlers.Following.IRI(p))
			err = deleteCollectionFromPath(r, handlers.Liked.IRI(p))
			return err
		})
	}
	if pub.ObjectTypes.Contains(it.GetType()) {
		return pub.OnObject(it, func(o *pub.Object) error {
			var err error
			err = deleteCollectionFromPath(r, handlers.Replies.IRI(o))
			err = deleteCollectionFromPath(r, handlers.Likes.IRI(o))
			err = deleteCollectionFromPath(r, handlers.Shares.IRI(o))
			return err
		})
	}
	return nil
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

func save(r *repo, it pub.Item) (pub.Item, error) {
	itPath := r.itemPath(it.GetLink())
	mkDirIfNotExists(itPath)

	if err := createCollections(*r, it); err != nil {
		return it, errors.Annotatef(err, "could not create object's collections")
	}
	// TODO(marius): it's possible to set the encoding/decoding functions on the package or storage object level
	//  instead of using jsonld.(Un)Marshal like this.
	entryBytes, err := encodeFn(it)
	if err != nil {
		return it, errors.Annotatef(err, "could not marshal object")
	}

	if err := mkDirIfNotExists(itPath); err != nil {
		r.errFn("unable to create path: %s, %s", itPath, err)
		return it, errors.Annotatef(err, "could not create file")
	}
	objPath := getObjectKey(itPath)
	f, err := os.OpenFile(objPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return it, err
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

	itPath := r.itemPath(col)
	err := fn(itPath)
	if err != nil {
		return errors.Annotatef(err, "Unable to save entries to collection %s", itPath)
	}
	return err
}

func loadRawFromPath(itPath string) ([]byte, error) {
	return ioutil.ReadFile(itPath)
}

func loadFromRaw(raw []byte) (pub.Item, error) {
	if raw == nil || len(raw) == 0 {
		// TODO(marius): log this instead of stopping the iteration and returning an error
		return nil, errors.Errorf("empty raw item")
	}
	return pub.UnmarshalJSON(raw)
}

func (r repo) loadOneFromPath(f s.Filterable) (pub.Item, error) {
	col, cnt, err := r.loadFromPath(f)
	if err != nil {
		return nil, err
	}
	if cnt == 0 {
		return nil, errors.NotFoundf("nothing found")
	}
	return col.First(), nil
}

func (r repo) loadItem(p string, f s.Filterable) (pub.Item, error) {
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

func (r repo) loadFromPath(f s.Filterable) (pub.ItemCollection, uint, error) {
	var err error
	col := make(pub.ItemCollection, 0)

	itPath := r.itemPath(f.GetLink())
	if isStorageCollectionKey(itPath) || itPath == r.path {
		err = filepath.Walk(itPath, func(p string, info os.FileInfo, err error) error {
			if err != nil && os.IsNotExist(err) {
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
			it, _ := r.loadItem(getObjectKey(p), f)
			if it != nil {
				col = append(col, it)
			}
			return nil
		})
	} else {
		var it pub.Item
		it, err = r.loadItem(getObjectKey(itPath), f)
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
