//go:build storage_fs || storage_all || (!storage_boltdb && !storage_badger && !storage_pgx && !storage_sqlite) || !go1.17

package fs

import (
	"crypto"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/fs"
	"io/ioutil"
	"math/rand"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/cache"
	"github.com/go-ap/fedbox/storage"
	"github.com/go-ap/processing"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ed25519"
)

var encodeFn = vocab.MarshalJSON
var decodeItemFn = vocab.UnmarshalJSON

var encodeMetadataFn = json.Marshal
var decodeMetadataFn = func(data []byte) (*storage.Metadata, error) {
	m := new(storage.Metadata)
	err := json.Unmarshal(data, m)
	return m, err
}

var errNotImplemented = errors.NotImplementedf("not implemented")

type loggerFn func(string, ...interface{})

var defaultLogFn = func(string, ...interface{}) {}

type Config struct {
	EnableCache bool
	StoragePath string
	BaseURL     string
}

// New returns a new repo repository
func New(c Config) (*repo, error) {
	p, err := getAbsStoragePath(c.StoragePath)
	if err != nil {
		return nil, err
	}
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
		cache:   cache.New(c.EnableCache),
	}
	return &b, nil
}

type repo struct {
	baseURL string
	path    string
	cwd     string
	opened  bool
	cache   cache.CanStore
	logFn   loggerFn
	errFn   loggerFn
}

// Open
func (r *repo) Open() error {
	if r.opened {
		return nil
	}
	return os.Chdir(r.path)
}

// Close
func (r *repo) Close() error {
	if r.opened {
		return nil
	}
	return os.Chdir(r.cwd)
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
		r.logFn("%s %s: %s", op, it.GetType(), it.GetLink())
	}
	return err
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

	ret, err := r.loadFromPath(f)
	if len(ret) == 1 && f.IsItemIRI() {
		return ret.First(), err
	}
	return ret, err
}

// Create
func (r *repo) Create(col vocab.CollectionInterface) (vocab.CollectionInterface, error) {
	if vocab.IsNil(col) {
		return col, errors.Newf("Unable to operate on nil element")
	}
	if len(col.GetLink()) == 0 {
		return col, errors.Newf("Invalid collection, it does not have a valid IRI")
	}
	return col, mkDirIfNotExists(r.itemPath(col.GetLink()))
}

// Save
func (r *repo) Save(it vocab.Item) (vocab.Item, error) {
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

// RemoveFrom
func (r *repo) RemoveFrom(col vocab.IRI, it vocab.Item) error {
	err := r.Open()
	defer r.Close()
	if err != nil {
		return err
	}

	ob, t := vocab.Split(col)
	var link vocab.IRI
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
	err = onCollection(r, col, it, func(p string) error {
		inCollection := false
		if fileInfo, err := ioutil.ReadDir(p); err == nil {
			for _, fi := range fileInfo {
				if fi.Name() == name && (isSymLink(fi) || isHardLink(fi)) {
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
	if err != nil {
		return err
	}
	r.cache.Remove(it.GetLink())
	return nil
}

func isSymLink(fi os.FileInfo) bool {
	return fi.Mode()&os.ModeSymlink == os.ModeSymlink
}

func isHardLink(fi os.FileInfo) bool {
	nlink := uint64(0)
	if sys := fi.Sys(); sys != nil {
		if stat, ok := sys.(*syscall.Stat_t); ok {
			nlink = uint64(stat.Nlink)
		}
	}
	return nlink > 1 && !fi.IsDir()
}

var allStorageCollections = append(vocab.ActivityPubCollections, ap.FedBOXCollections...)

// AddTo
func (r *repo) AddTo(col vocab.IRI, it vocab.Item) error {
	err := r.Open()
	defer r.Close()
	if err != nil {
		return err
	}

	ob, t := allStorageCollections.Split(col)
	var link vocab.IRI
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
				if fi.Name() == fullLink && (isSymLink(fi) || isHardLink(fi)) {
					inCollection = true
				}
			}
		}
		if inCollection {
			return nil
		}

		if itPath, err = filepath.Abs(itPath); err != nil {
			return err
		}
		if fullLink, err = filepath.Abs(fullLink); err != nil {
			return err
		}
		if itPath, err = filepath.Rel(fullLink, itPath); err != nil {
			return err
		}
		// NOTE(marius): using filepath.Rel returns one extra parent for some reason, I need to look into why
		itPath = strings.Replace(itPath, "../", "", 1)

		// NOTE(marius): we can't use hard links as we're linking to folders :(
		// This would have been tremendously easier (as in, not having to compute paths) with hard-links.
		return os.Symlink(itPath, fullLink)
	})
}

// Delete
func (r *repo) Delete(it vocab.Item) error {
	err := r.Open()
	defer r.Close()
	if err != nil {
		return err
	}
	return delete(r, it)
}

func delete(r *repo, it vocab.Item) error {
	if it.IsCollection() {
		return vocab.OnCollectionIntf(it, func(c vocab.CollectionInterface) error {
			var err error
			for _, it := range c.Collection() {
				if err = deleteItem(r, it); err != nil {
					r.logFn("Unable to remove item %s", it.GetLink())
				}
			}
			return nil
		})
	}
	return deleteItem(r, it.GetLink())
}

// PasswordSet
func (r *repo) PasswordSet(it vocab.Item, pw []byte) error {
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
func (r *repo) PasswordCheck(it vocab.Item, pw []byte) error {
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
func (r *repo) LoadMetadata(iri vocab.IRI) (*storage.Metadata, error) {
	err := r.Open()
	defer r.Close()
	if err != nil {
		return nil, err
	}

	p := r.itemPath(iri)
	raw, err := loadRawFromPath(getMetadataKey(p))
	if err != nil {
		return nil, errors.NewNotFound(r.asPathErr(err), "Could not find metadata in path %s", p)
	}
	m, err := decodeMetadataFn(raw)
	if err != nil {
		return nil, errors.Annotatef(err, "Could not unmarshal metadata")
	}
	return m, nil
}

// SaveMetadata
func (r *repo) SaveMetadata(m storage.Metadata, iri vocab.IRI) error {
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

	entryBytes, err := encodeMetadataFn(m)
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

// LoadKey loads a private key for an actor found by its IRI
func (r *repo) LoadKey(iri vocab.IRI) (crypto.PrivateKey, error) {
	m, err := r.LoadMetadata(iri)
	if err != nil {
		return nil, r.asPathErr(err)
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

// GenKey creates and saves a private key for an actor found by its IRI
func (r *repo) GenKey(iri vocab.IRI) error {
	ob, err := r.loadOneFromPath(iri)
	if err != nil {
		return err
	}
	if ob.GetType() != vocab.PersonType {
		return errors.Newf("trying to generate keys for invalid ActivityPub object type: %s", ob.GetType())
	}
	m, err := r.LoadMetadata(iri)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	if m == nil {
		m = new(storage.Metadata)
	}
	if m.PrivateKey != nil {
		return nil
	}
	// TODO(marius): this needs a way to choose between ED25519 and RSA keys
	pubB, prvB := generateECKeyPair()
	m.PrivateKey = pem.EncodeToMemory(&prvB)

	if err = r.SaveMetadata(*m, iri); err != nil {
		return err
	}
	vocab.OnActor(ob, func(act *vocab.Actor) error {
		act.PublicKey = vocab.PublicKey{
			ID:           vocab.IRI(fmt.Sprintf("%s#main", iri)),
			Owner:        iri,
			PublicKeyPem: string(pem.EncodeToMemory(&pubB)),
		}
		return nil
	})
	return nil
}

func generateECKeyPair() (pem.Block, pem.Block) {
	// TODO(marius): make this actually produce proper keys
	keyPub, keyPrv, _ := ed25519.GenerateKey(rand.New(rand.NewSource(6667)))

	var p, r pem.Block
	if pubEnc, err := x509.MarshalPKIXPublicKey(keyPub); err == nil {
		p = pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: pubEnc,
		}
	}
	if prvEnc, err := x509.MarshalPKCS8PrivateKey(keyPrv); err == nil {
		r = pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: prvEnc,
		}
	}
	return p, r
}

func createOrOpenFile(p string) (*os.File, error) {
	err := mkDirIfNotExists(path.Dir(p))
	if err != nil {
		return nil, err
	}
	return os.OpenFile(p, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
}

var storageCollectionPaths = append(ap.FedBOXCollections, append(vocab.OfActor, vocab.OfObject...)...)

func isStorageCollectionKey(p string) bool {
	lst := vocab.CollectionPath(path.Base(p))
	return storageCollectionPaths.Contains(lst)
}

func (r repo) itemPath(iri vocab.IRI) string {
	url, err := iri.URL()
	if err != nil {
		return ""
	}
	return path.Join(r.path, url.Host, url.Path)
}

// createCollections
func createCollections(r repo, it vocab.Item) error {
	if vocab.IsNil(it) || !it.IsObject() {
		return nil
	}
	if vocab.ActorTypes.Contains(it.GetType()) {
		vocab.OnActor(it, func(p *vocab.Actor) error {
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
	return vocab.OnObject(it, func(o *vocab.Object) error {
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

const (
	objectKey   = "__raw.json"
	metaDataKey = "__meta_data.json"
)

func getMetadataKey(p string) string {
	return path.Join(p, metaDataKey)
}

func getObjectKey(p string) string {
	return path.Join(p, objectKey)
}

func createCollectionInPath(r repo, it vocab.Item) (vocab.Item, error) {
	itPath := r.itemPath(it.GetLink())
	return it.GetLink(), r.asPathErr(mkDirIfNotExists(itPath))
}

func deleteCollectionFromPath(r repo, it vocab.Item) error {
	if vocab.IsNil(it) {
		return nil
	}
	itPath := r.itemPath(it.GetLink())
	if fi, err := os.Stat(itPath); err != nil {
		if !os.IsNotExist(err) {
			return errors.NewNotFound(r.asPathErr(err), "not found")
		}
	} else if fi.IsDir() {
		return os.Remove(itPath)
	}
	r.cache.Remove(it.GetLink())
	return nil
}

// deleteCollections
func deleteCollections(r repo, it vocab.Item) error {
	if vocab.ActorTypes.Contains(it.GetType()) {
		return vocab.OnActor(it, func(p *vocab.Actor) error {
			var err error
			err = deleteCollectionFromPath(r, vocab.Inbox.IRI(p))
			err = deleteCollectionFromPath(r, vocab.Outbox.IRI(p))
			err = deleteCollectionFromPath(r, vocab.Followers.IRI(p))
			err = deleteCollectionFromPath(r, vocab.Following.IRI(p))
			err = deleteCollectionFromPath(r, vocab.Liked.IRI(p))
			return err
		})
	}
	if vocab.ObjectTypes.Contains(it.GetType()) {
		return vocab.OnObject(it, func(o *vocab.Object) error {
			var err error
			err = deleteCollectionFromPath(r, vocab.Replies.IRI(o))
			err = deleteCollectionFromPath(r, vocab.Likes.IRI(o))
			err = deleteCollectionFromPath(r, vocab.Shares.IRI(o))
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

func deleteItem(r *repo, it vocab.Item) error {
	itemPath := r.itemPath(it.GetLink())
	if err := os.RemoveAll(itemPath); err != nil {
		return err
	}
	r.cache.Remove(it.GetLink())
	return nil
}

func save(r *repo, it vocab.Item) (vocab.Item, error) {
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
		return it, errors.NewNotFound(err, "%s not found", objPath)
	}
	defer f.Close()
	wrote, err := f.Write(entryBytes)
	if err != nil {
		return it, errors.Annotatef(err, "could not store encoded object")
	}
	if wrote != len(entryBytes) {
		return it, errors.Annotatef(err, "failed writing object")
	}

	r.cache.Set(it.GetLink(), it)
	return it, nil
}

// IsLocalIRI shows if the received IRI belongs to the current instance
func (r repo) IsLocalIRI(i vocab.IRI) bool {
	return i.Contains(vocab.IRI(r.baseURL), false)
}

func onCollection(r *repo, col vocab.IRI, it vocab.Item, fn func(p string) error) error {
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

	itPath := r.itemPath(col)
	err := fn(itPath)
	if err != nil {
		if os.IsExist(err) {
			return errors.NewConflict(err, "%s already exists in collection %s", it.GetID(), itPath)
		}
		return errors.Annotatef(err, "Unable to save entries to collection %s", itPath)
	}
	return nil
}

func loadRawFromPath(itPath string) ([]byte, error) {
	return ioutil.ReadFile(itPath)
}

func loadFromRaw(raw []byte) (vocab.Item, error) {
	if raw == nil || len(raw) == 0 {
		// TODO(marius): log this instead of stopping the iteration and returning an error
		return nil, errors.Errorf("empty raw item")
	}
	return decodeItemFn(raw)
}

func (r repo) loadOneFromPath(f processing.Filterable) (vocab.Item, error) {
	col, err := r.loadFromPath(f)
	if err != nil {
		return nil, err
	}
	if col == nil {
		return nil, errors.NotFoundf("nothing found")
	}
	if col.IsCollection() {
		var result vocab.Item
		vocab.OnCollectionIntf(col, func(col vocab.CollectionInterface) error {
			result = col.Collection().First()
			return nil
		})
		return result, nil
	}
	return col, nil
}

func isSingleItem(f processing.Filterable) bool {
	if _, isIRI := f.(vocab.IRI); isIRI {
		return true
	}
	if _, isItem := f.(vocab.Item); isItem {
		return true
	}
	return false
}

func loadFilteredPropsForActor(r repo, f processing.Filterable) func(a *vocab.Actor) error {
	return func(a *vocab.Actor) error {
		return vocab.OnObject(a, loadFilteredPropsForObject(r, f))
	}
}

var subFilterValidationError = errors.NotValidf("subfilter failed validation")

func loadFilteredPropsForObject(r repo, f processing.Filterable) func(o *vocab.Object) error {
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

func loadFilteredPropsForActivity(r repo, f processing.Filterable) func(a *vocab.Activity) error {
	return func(a *vocab.Activity) error {
		if ok, fo := ap.FiltersOnActivityObject(f); ok && !vocab.IsNil(a.Object) && vocab.IsIRI(a.Object) {
			if ob, err := r.loadOneFromPath(a.Object.GetLink()); err == nil {
				a.Object, _ = ap.FilterIt(ob, fo)
			}
			if a.Object == nil {
				return subFilterValidationError
			}
		}
		return vocab.OnIntransitiveActivity(a, loadFilteredPropsForIntransitiveActivity(r, f))
	}
}

func loadFilteredPropsForIntransitiveActivity(r repo, f processing.Filterable) func(a *vocab.IntransitiveActivity) error {
	return func(a *vocab.IntransitiveActivity) error {
		if ok, fa := ap.FiltersOnActivityActor(f); ok && !vocab.IsNil(a.Actor) && vocab.IsIRI(a.Actor) {
			if act, err := r.loadOneFromPath(a.Actor.GetLink()); err == nil {
				a.Actor, _ = ap.FilterIt(act, fa)
			}
			if a.Actor == nil {
				return subFilterValidationError
			}
		}
		if ok, ft := ap.FiltersOnActivityTarget(f); ok && !vocab.IsNil(a.Target) && vocab.IsIRI(a.Target) {
			if t, err := r.loadOneFromPath(a.Target.GetLink()); err == nil {
				a.Target, _ = ap.FilterIt(t, ft)
			}
			if a.Target == nil {
				return subFilterValidationError
			}
		}
		return vocab.OnObject(a, loadFilteredPropsForObject(r, f))
	}
}

func (r repo) asPathErr(err error) error {
	if perr, ok := err.(*fs.PathError); ok {
		p := strings.TrimPrefix(perr.Path, r.path)
		p = strings.TrimSuffix(p, objectKey)
		p = strings.TrimSuffix(p, metaDataKey)
		perr.Path = strings.Trim(p, "/")
		return perr
	}
	return err
}

func getOriginalIRI(p string) (vocab.Item, error) {
	// NOTE(marius): if the __raw file wasn't found, but the path corresponds to a valid symlink,
	// we can interpret that as an IRI (usually referencing an external object) and return that.
	dir := path.Dir(p)
	fi, err := os.Stat(dir)
	if err != nil {
		return nil, nil
	}
	if !fi.IsDir() {
		return nil, nil
	}
	original, err := os.Readlink(dir)
	if err != nil {
		return nil, nil
	}
	original = strings.TrimLeft(path.Clean(original), "../")
	pieces := strings.Split(original, "/")
	if len(pieces) == 0 {
		return nil, nil
	}
	upath := ""
	host := pieces[0]
	if len(pieces) > 1 {
		upath = path.Join(pieces[1:]...)
	}
	u := url.URL{Scheme: "https", Host: host, Path: upath}
	return vocab.IRI(u.String()), nil
}

func (r repo) loadItem(p string, f processing.Filterable) (vocab.Item, error) {
	var it vocab.Item
	if cachedIt := r.cache.Get(f.GetLink()); cachedIt != nil {
		it = cachedIt
	}
	if vocab.IsNil(it) {
		raw, err := loadRawFromPath(p)
		if err != nil {
			if os.IsNotExist(err) {
				return getOriginalIRI(p)
			}
			return nil, r.asPathErr(err)
		}
		if raw == nil {
			return nil, nil
		}
		it, err = loadFromRaw(raw)
		if err != nil {
			return nil, r.asPathErr(err)
		}
		if vocab.IsNil(it) {
			return nil, errors.NotFoundf("not found")
		}
	}
	if it.IsCollection() {
		// we need to dereference them, so no further filtering/processing is needed here
		return it, nil
	}
	if vocab.IsIRI(it) {
		if it, _ = r.loadOneFromPath(it.GetLink()); vocab.IsNil(it) {
			return nil, errors.NotFoundf("not found")
		}
	}
	typ := it.GetType()
	// NOTE(marius): this can probably expedite filtering if we early exit if we fail to load the
	// properties that need to load for sub-filters.
	if vocab.IntransitiveActivityTypes.Contains(typ) {
		if validErr := vocab.OnIntransitiveActivity(it, loadFilteredPropsForIntransitiveActivity(r, f)); validErr != nil {
			return nil, nil
		}
	}
	if vocab.ActivityTypes.Contains(typ) {
		if validErr := vocab.OnActivity(it, loadFilteredPropsForActivity(r, f)); validErr != nil {
			return nil, nil
		}
	}
	if vocab.ActorTypes.Contains(typ) {
		if validErr := vocab.OnActor(it, loadFilteredPropsForActor(r, f)); validErr != nil {
			return nil, nil
		}
	}
	if vocab.ObjectTypes.Contains(typ) {
		if validErr := vocab.OnObject(it, loadFilteredPropsForObject(r, f)); validErr != nil {
			return nil, nil
		}
	}

	r.cache.Set(it.GetLink(), it)
	if f != nil {
		return ap.FilterIt(it, f)
	}
	return it, nil
}

func (r repo) loadFromPath(f processing.Filterable) (vocab.ItemCollection, error) {
	var err error
	col := make(vocab.ItemCollection, 0)

	itPath := r.itemPath(f.GetLink())
	if isStorageCollectionKey(itPath) || itPath == r.path {
		err = filepath.Walk(itPath, func(p string, info os.FileInfo, err error) error {
			if err != nil && os.IsNotExist(err) {
				if isStorageCollectionKey(p) {
					return errors.NewNotFound(r.asPathErr(err), "not found")
				}
				r.errFn("Error when loading path %s: %s", p, err)
				return nil
			}
			dirPath, _ := path.Split(p)
			dir := strings.TrimRight(dirPath, "/")
			if dir != itPath {
				return nil
			}
			if _, ok := f.(vocab.IRI); ok {
				// when loading a collection by path, we want to avoid filtering out IRIs that don't specifically
				// contain the path, so we set the filter to a nil value
				f = nil
			}
			if it, _ := r.loadItem(getObjectKey(p), f); !vocab.IsNil(it) {
				col = append(col, it)
			}
			return nil
		})
	} else {
		it, err := r.loadItem(getObjectKey(itPath), f)
		if err != nil {
			return nil, errors.NewNotFound(err, "not found")
		}
		if !vocab.IsNil(it) {
			col = append(col, it)
		}
	}
	return col, err
}
