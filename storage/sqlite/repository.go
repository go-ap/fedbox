// +build storage_sqlite storage_all !sqlite_fs,!storage_boltdb,!storage_badger,!storage_pgx

package sqlite

import (
	"database/sql"
	"fmt"
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/storage"
	"github.com/go-ap/jsonld"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

var encodeFn = jsonld.Marshal
var decodeFn = jsonld.Unmarshal

var errNotImplemented = errors.NotImplementedf("not implemented")

type loggerFn func(string, ...interface{})

var defaultLogFn = func(string, ...interface{}) {}

type Config struct {
	StoragePath string
	Env 		string
	BaseURL		string
}

// New returns a new repo repository
func New(c Config) (*repo, error) {
	p, err := getAbsStoragePath(c.StoragePath)
	if err != nil {
		return nil, err
	}
	p = path.Clean(path.Join(p, c.Env))
	if err := mkDirIfNotExists(p); err != nil {
		return nil, err
	}
	host := url.PathEscape(c.BaseURL)
	if u, err := url.Parse(c.BaseURL); err == nil {
		host = u.Host
	}
	p = fmt.Sprintf("%s/%s.sqlite", p, host)
	b := repo{
		path:    p,
		baseURL: c.BaseURL,
		logFn:   defaultLogFn,
		errFn:   defaultLogFn,
	}
	return &b, nil
}

type repo struct {
	conn    *sql.DB
	baseURL string
	path    string
	logFn   loggerFn
	errFn   loggerFn
}

// Open
func (r *repo) Open() error {
	var err error
	r.conn, err = sql.Open("sqlite", r.path)
	return err
}

// Close
func (r *repo) Close() error {
	return r.conn.Close()
}

func (r repo) CreateService(service pub.Service) error {
	err := r.Open()
	defer r.Close()
	if err != nil {
		return err
	}
	it, err := save(r, service)
	if err != nil {
		r.errFn("%s %s: %s", err, it.GetType(), it.GetLink())
	}
	return err
}

// Load
func (r *repo) Load(i pub.IRI) (pub.Item, error) {
	return nil, errNotImplemented
}

// Save
func (r *repo) Save(it pub.Item) (pub.Item, error) {
	return nil, errNotImplemented
}

// Create
func (r *repo) Create(col pub.CollectionInterface) (pub.CollectionInterface, error) {
	return nil, errNotImplemented
}

// RemoveFrom
func (r *repo) RemoveFrom(col pub.IRI, it pub.Item) error {
	return errNotImplemented
}

// AddTo
func (r *repo) AddTo(col pub.IRI, it pub.Item) error {
	return errNotImplemented
}

// Delete
func (r *repo) Delete(it pub.Item) (pub.Item, error) {
	return nil, errNotImplemented
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

func save(l repo, it pub.Item) (pub.Item, error) {
	table := string(ap.ObjectsType)
	if pub.ActivityTypes.Contains(it.GetType()) {
		table = string(ap.ActivitiesType)
	} else if pub.ActorTypes.Contains(it.GetType()) {
		table = string(ap.ActorsType)
	}
	query := fmt.Sprintf("INSERT INTO %s (key, iri, created_at, type, raw) VALUES ($1, $2, $3, $4, $5);", table)

	iri := it.GetLink()
	uuid := path.Base(iri.String())
	if uuid == "." {
		// broken ID generation
		return it, errors.Newf("Unable to get ID for %s[%s]", table, it.GetType())
	}
	raw, _ := encodeFn(it)
	_, err := l.conn.Exec(query, uuid, iri, time.Now(), it.GetType(), raw)
	if err != nil {
		l.errFn("query error: %s", err)
		return it, errors.Annotatef(err, "query error")
	}

	return it, nil
}
