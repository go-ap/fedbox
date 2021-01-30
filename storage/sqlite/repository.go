// +build storage_sqlite storage_all !sqlite_fs,!storage_boltdb,!storage_badger,!storage_pgx

package sqlite

import (
	"database/sql"
	"fmt"
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/storage"
	"github.com/go-ap/handlers"
	"github.com/go-ap/jsonld"
	s "github.com/go-ap/storage"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
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
	p, err := getFullPath(c)
	return &repo{
		path:    p,
		baseURL: c.BaseURL,
		logFn:   defaultLogFn,
		errFn:   defaultLogFn,
	}, err
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

func getCollectionTable(typ handlers.CollectionType) string {
	switch typ {
	case handlers.Followers:
		fallthrough
	case handlers.Following:
		fallthrough
	case "actors", "":
		return "actors"
	case handlers.Inbox:
		fallthrough
	case handlers.Outbox:
		fallthrough
	case handlers.Shares:
		fallthrough
	case handlers.Liked:
		fallthrough
	case handlers.Likes:
		fallthrough
	case "activities":
		return "activities"
	case handlers.Replies:
		fallthrough
	default:
		return "objects"
	}
	return "objects"
}

// Load
func (r *repo) Load(i pub.IRI) (pub.Item, error) {
	f, err := ap.FiltersFromIRI(i)
	if err != nil {
		return nil, err
	}
	if err = r.Open(); err != nil {
		return nil, err
	}
	defer r.Close()
	return loadFromDb(r.conn, getCollectionTable(f.Collection), f)
}

// Save
func (r *repo) Save(it pub.Item) (pub.Item, error) {
	if err := r.Open(); err != nil {
		return nil, err
	}
	defer r.Close()
	return save(*r, it)
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

func getFullPath(c Config) (string, error) {
	p, err := getAbsStoragePath(c.StoragePath)
	if err != nil {
		return "memory", err
	}
	p = path.Clean(path.Join(p, c.Env))
	if err := mkDirIfNotExists(p); err != nil {
		return "memory", err
	}
	host := url.PathEscape(c.BaseURL)
	if u, err := url.Parse(c.BaseURL); err == nil {
		host = u.Host
	}
	return fmt.Sprintf("%s/%s.sqlite", p, host), nil
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

func loadFromDb(conn *sql.DB, table string, f s.Filterable) (pub.Item, error) {
	clauses, values := getWhereClauses(f)
	var total uint = 0

	sel := fmt.Sprintf("SELECT id, iri, published, type, raw FROM %s WHERE %s ORDER BY published %s", table, strings.Join(clauses, " AND "), getLimit(f))
	rows, err := conn.Query(sel, values...)
	if err != nil {
		if err == sql.ErrNoRows {
			return pub.ItemCollection{}, nil
		}
		return nil, errors.Annotatef(err, "unable to run select")
	}

	ret := make(pub.ItemCollection, 0)
	// Iterate through the result set
	for rows.Next() {
		var id int64
		var iri string
		var created string
		var typ string
		var raw []byte
		err = rows.Scan(&id, &iri, &created, &typ, &raw)
		if err != nil {
			return ret, errors.Annotatef(err, "scan values error")
		}

		it, err := pub.UnmarshalJSON(raw)
		if err != nil {
			return ret, errors.Annotatef(err, "unable to unmarshal raw item")
		}
		ret = append(ret, it)
	}

	selCnt := fmt.Sprintf("SELECT COUNT(id) FROM %s WHERE %s", table, strings.Join(clauses, " AND "))
	if err = conn.QueryRow(selCnt, values...).Scan(&total); err != nil {
		err = errors.Annotatef(err, "unable to count all rows")
	}

	return ret, err
}

func save(l repo, it pub.Item) (pub.Item, error) {
	table := string(ap.ObjectsType)
	if pub.ActivityTypes.Contains(it.GetType()) {
		table = string(ap.ActivitiesType)
	} else if pub.ActorTypes.Contains(it.GetType()) {
		table = string(ap.ActorsType)
	}
	query := fmt.Sprintf("INSERT INTO %s (iri, published, type, raw) VALUES (?, ?, ?, ?);", table)

	iri := it.GetLink()
	raw, err := encodeFn(it)
	if err != nil {
		l.errFn("query error: %s\n%s\n%#v", err, query)
		return it, errors.Annotatef(err, "query error")
	}
	if _, err = l.conn.Exec(query, iri, time.Now().UTC(), it.GetType(), raw); err != nil {
		l.errFn("query error: %s\n%s\n%#v", err, query)
		return it, errors.Annotatef(err, "query error")
	}

	return it, nil
}
