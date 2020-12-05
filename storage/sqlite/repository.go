package sqlite

import (
	"database/sql"
	"fmt"
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/storage"
	"github.com/go-ap/jsonld"
	s "github.com/go-ap/storage"
	"os"
	"path"
	"path/filepath"
)

var encodeFn = jsonld.Marshal
var decodeFn = jsonld.Unmarshal

var errNotImplemented = errors.NotImplementedf("not implemented")

type loggerFn func(string, ...interface{})

var defaultLogFn = func(string, ...interface{}) {}

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
	db, err := sql.Open("sqlite", "")
	b := repo{
		db: db,
		path:    p,
		baseURL: c.BaseURL,
		logFn:   defaultLogFn,
		errFn:   defaultLogFn,
	}
	return &b, nil
}

type repo struct {
	db      *sql.DB
	baseURL string
	path    string
	logFn   loggerFn
	errFn   loggerFn
}

// Open
func (r *repo) Open() error {
	return errNotImplemented
}

// Close
func (r *repo) Close() error {
	return errNotImplemented
}

func (r *repo) CreateService(service pub.Service) error {
	return errNotImplemented
}

// LoadActivities
func (r *repo) LoadActivities(f s.Filterable) (pub.ItemCollection, uint, error) {
	return nil, 0, errNotImplemented
}
// LoadObjects
func (r *repo) LoadObjects(f s.Filterable) (pub.ItemCollection, uint, error) {
	return nil, 0, errNotImplemented
}

// LoadActors
func (r *repo) LoadActors(f s.Filterable) (pub.ItemCollection, uint, error) {
	return nil, 0, errNotImplemented
}

// LoadCollection
func (r *repo) LoadCollection(f s.Filterable) (pub.CollectionInterface, error) {
	return nil, errNotImplemented
}

// CreateCollection
func (r *repo) CreateCollection(col pub.CollectionInterface) (pub.CollectionInterface, error) {
	return nil, errNotImplemented
}

// SaveActivity
func (r *repo) SaveActivity(it pub.Item) (pub.Item, error) {
	return nil, errNotImplemented
}

// SaveActor
func (r *repo) SaveActor(it pub.Item) (pub.Item, error) {
	return nil, errNotImplemented
}

// SaveObject
func (r *repo) SaveObject(it pub.Item) (pub.Item, error) {
	return nil, errNotImplemented
}

// RemoveFromCollection
func (r *repo) RemoveFromCollection(col pub.IRI, it pub.Item) error {
	return errNotImplemented
}

// AddToCollection
func (r *repo) AddToCollection(col pub.IRI, it pub.Item) error {
	return errNotImplemented
}

// UpdateActor
func (r *repo) UpdateActor(it pub.Item) (pub.Item, error) {
	return nil, errNotImplemented
}

// UpdateObject
func (r *repo) UpdateObject(it pub.Item) (pub.Item, error) {
	return nil, errNotImplemented
}

// DeleteActor
func (r *repo) DeleteActor(it pub.Item) (pub.Item, error) {
	return nil, errNotImplemented
}

// DeleteObject
func (r *repo) DeleteObject(it pub.Item) (pub.Item, error) {
	return nil, errNotImplemented
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
	return errNotImplemented
}

// PasswordCheck
func (r *repo) PasswordCheck(it pub.Item, pw []byte) error {
	return errNotImplemented
}

// LoadMetadata
func (r *repo) LoadMetadata(iri pub.IRI) (*storage.Metadata, error) {
	return nil, errNotImplemented
}

// SaveMetadata
func (r *repo) SaveMetadata(m storage.Metadata, iri pub.IRI) error {
	return errNotImplemented
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
