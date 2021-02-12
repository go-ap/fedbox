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
	"golang.org/x/crypto/bcrypt"
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
func getCollectionTypeFromIRI(i string) handlers.CollectionType {
	col := handlers.CollectionType(path.Base(i))
	if !ap.FedboxCollections.Contains(col) {
		b, _ := path.Split(i)
		col = handlers.CollectionType(path.Base(b))
	}
	switch col {
	case handlers.Followers:
		fallthrough
	case handlers.Following:
		fallthrough
	case "actors":
		return "actors"
	case handlers.Inbox:
		fallthrough
	case handlers.Outbox:
		fallthrough
	case handlers.Shares:
		fallthrough
	case handlers.Likes:
		fallthrough
	case "activities":
		return "activities"
	case handlers.Liked:
		fallthrough
	case handlers.Replies:
		fallthrough
	default:
		return "objects"
	}
	return col
}

func getCollectionTable(f *ap.Filters) string {
	typ := f.Collection
	switch typ {
	case handlers.Followers:
		fallthrough
	case handlers.Following:
		fallthrough
	case "actors":
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
	return loadFromDb(r.conn, f)
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
	if col.IsObject() {
		_, err := r.Save(col)
		if err != nil {
			return col, err
		}
	}
	return col, nil
}

// RemoveFrom
func (r *repo) RemoveFrom(col pub.IRI, it pub.Item) error {
	return errNotImplemented
}

// AddTo
func (r *repo) AddTo(col pub.IRI, it pub.Item) error {
	if err := r.Open(); err != nil {
		return err
	}
	defer r.Close()
	query := "INSERT INTO collections (iri, object) VALUES (?, ?);"

	if _, err := r.conn.Exec(query, col, it.GetLink()); err != nil {
		r.errFn("query error: %s\n%s\n%#v", err, query)
		return errors.Annotatef(err, "query error")
	}

	return nil
}

// Delete
func (r *repo) Delete(it pub.Item) (pub.Item, error) {
	err := r.Open()
	defer r.Close()
	if err != nil {
		return nil, err
	}

	if it.IsCollection() {
		err := pub.OnCollectionIntf(it, func(c pub.CollectionInterface) error {
			var err error
			for _, it := range c.Collection() {
				if it, err = r.Delete(it); err != nil {
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
		if old, err := loadFromOneTable(r.conn, f); err == nil {
			t.FormerType = old.GetType()
		}
	}

	//deleteCollections(*r, it)
	return save(*r, t)
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
	if err != nil {
		return nil, err
	}
	defer r.Close()

	m := new(storage.Metadata)
	raw, err := loadMetadataFromTable(r.conn, iri)
	if err != nil {
		return nil, err
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
	if err != nil {
		return err
	}
	defer r.Close()

	entryBytes, err := encodeFn(m)
	if err != nil {
		return errors.Annotatef(err, "Could not marshal metadata")
	}
	return saveMetadataToTable(r.conn, iri, entryBytes)
}

func getFullPath(c Config) (string, error) {
	p, err := getAbsStoragePath(c.StoragePath)
	if err != nil {
		return "memory", err
	}
	if err := mkDirIfNotExists(p); err != nil {
		return "memory", err
	}
	return path.Join(p, "storage.sqlite"), nil
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

func saveMetadataToTable(conn *sql.DB, iri pub.IRI, m []byte) error {
	table := getCollectionTypeFromIRI(iri.String())

	query := fmt.Sprintf("UPDATE %s SET meta = ? WHERE iri = ?;", table)
	_, err := conn.Exec(query, m, iri)
	return err
}

func loadMetadataFromTable(conn *sql.DB, iri pub.IRI) ([]byte, error) {
	table := getCollectionTypeFromIRI(iri.String())

	var meta []byte
	sel := fmt.Sprintf("SELECT meta FROM %s WHERE iri = ?;", table)
	err := conn.QueryRow(sel, iri).Scan(&meta)
	return meta, err
}

func loadFromOneTable(conn *sql.DB, f *ap.Filters) (pub.ItemCollection, error) {
	table := getCollectionTable(f)
	clauses, values := getWhereClauses(f)
	var total uint = 0

	selCnt := fmt.Sprintf("SELECT COUNT(id) FROM %s WHERE %s", table, strings.Join(clauses, " AND "))
	if err := conn.QueryRow(selCnt, values...).Scan(&total); err != nil {
		return nil, errors.Annotatef(err, "unable to count all rows")
	}
	ret := make(pub.ItemCollection, 0)
	if total == 0 {
		return ret, nil
	}

	sel := fmt.Sprintf("SELECT id, iri, published, type, raw FROM %s WHERE %s ORDER BY published %s", table, strings.Join(clauses, " AND "), getLimit(f))
	rows, err := conn.Query(sel, values...)
	if err != nil {
		if err == sql.ErrNoRows {
			return pub.ItemCollection{}, nil
		}
		return nil, errors.Annotatef(err, "unable to run select")
	}

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

	return ret, err
}

func loadFromDb(conn *sql.DB, f *ap.Filters) (pub.Item, error) {
	table := getCollectionTable(f)
	clauses, values := getWhereClauses(f)
	var total uint = 0

	// todo(marius): this needs to be split into three cases:
	//  1. IRI corresponds to a collection that is not one of the storage tables (ie, not activities, actors, objects):
	//    Then we look for correspondences in the collections table.
	// 2. The IRI corresponds to the activities, actors, objects tables:
	//    Then we load from the corresponding table using `iri LIKE IRI%` criteria
	// 3. IRI corresponds to an object: we load directly from the corresponding table.
	selCnt := fmt.Sprintf("SELECT COUNT(id) FROM %s WHERE %s", table, strings.Join(clauses, " AND "))
	if err := conn.QueryRow(selCnt, values...).Scan(&total); err != nil && err != sql.ErrNoRows {
		return nil, errors.Annotatef(err, "unable to count all rows")
	}
	if total > 0 {
		return loadFromOneTable(conn, f)
	}
	var (
		iriClause string
		iriValue interface{}
		hasIRI = false
	)
	for i, c := range clauses {
		if strings.Contains(c, "iri") {
			iriClause = c
			iriValue = values[i]
			hasIRI = true
		}
	}
	if !hasIRI {
		return nil, errors.NotFoundf("Not found")
	}
	colCntQ := fmt.Sprintf("SELECT COUNT(id) FROM %s WHERE %s %s", "collections", iriClause, getLimit(f))
	if err := conn.QueryRow(colCntQ, iriValue).Scan(&total); err != nil && err != sql.ErrNoRows {
		return nil, errors.Annotatef(err, "unable to count all rows")
	}
	if total == 0 && handlers.ActivityPubCollections.Contains(f.Collection) && f.Collection != handlers.Inbox {
		return nil, errors.NotFoundf("Unable to find collection %s", f.Collection)
	}
	sel := fmt.Sprintf("SELECT id, iri, object FROM %s WHERE %s %s", "collections", iriClause, getLimit(f))
	rows, err := conn.Query(sel, iriValue)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NotFoundf("Unable to load %s", f.Collection)
		}
		return nil, errors.Annotatef(err, "unable to run select")
	}
	fOb := *f
	fActors := *f
	fActivities := *f

	fOb.IRI = ""
	fOb.Collection = "objects"
	fOb.ItemKey = make(ap.CompStrs, 0)
	fActors.IRI = ""
	fActors.Collection = "actors"
	fActors.ItemKey = make(ap.CompStrs, 0)
	fActivities.IRI = ""
	fActivities.Collection = "activities"
	fActivities.ItemKey = make(ap.CompStrs, 0)
	// Iterate through the result set
	for rows.Next() {
		var id int64
		var object string
		var iri string

		err = rows.Scan(&id, &iri, &object)
		if err != nil {
			return pub.ItemCollection{}, errors.Annotatef(err, "scan values error")
		}
		col := getCollectionTypeFromIRI(iri)
		if col == "objects" {
			fOb.ItemKey = append(f.ItemKey, ap.StringEquals(object))
		} else if col == "actors" {
			fActors.ItemKey = append(f.ItemKey, ap.StringEquals(object))
		} else if col == "activities" {
			fActivities.ItemKey = append(f.ItemKey, ap.StringEquals(object))
		} else {
			switch table {
			case "activities":
				fActivities.ItemKey = append(f.ItemKey, ap.StringEquals(object))
			case "actors":
				fActors.ItemKey = append(f.ItemKey, ap.StringEquals(object))
			case "objects":
				fallthrough
			default:
				fOb.ItemKey = append(f.ItemKey, ap.StringEquals(object))
			}
		}
	}
	ret := make(pub.ItemCollection, 0)
	if len(fActivities.ItemKey) > 0 {
		retAct, err := loadFromOneTable(conn, &fActivities)
		if err != nil {
			return ret, err
		}
		ret = append(ret, retAct...)
	}
	if len(fActors.ItemKey) > 0 {
		retAct, err := loadFromOneTable(conn, &fActors)
		if err != nil {
			return ret, err
		}
		ret = append(ret, retAct...)
	}
	if len(fOb.ItemKey) > 0 {
		retOb, err := loadFromOneTable(conn, &fOb)
		if err != nil {
			return ret, err
		}
		ret = append(ret, retOb...)
	}
	return ret, nil
}

func save(l repo, it pub.Item) (pub.Item, error) {
	iri := it.GetLink()

	table := string(ap.ObjectsType)
	if pub.ActivityTypes.Contains(it.GetType()) {
		table = string(ap.ActivitiesType)
	} else if pub.ActorTypes.Contains(it.GetType()) {
		table = string(ap.ActorsType)
	} else if it.GetType() == pub.TombstoneType {
		if strings.Contains(iri.String(), string(ap.ActorsType)) {
			table = string(ap.ActorsType)
		}
		if strings.Contains(iri.String(), string(ap.ActivitiesType)) {
			table = string(ap.ActivitiesType)
		}
	}
	query := fmt.Sprintf("INSERT OR REPLACE INTO %s (iri, published, type, raw) VALUES (?, ?, ?, ?) ;", table)

	raw, err := encodeFn(it)
	if err != nil {
		l.errFn("query error: %s\n%s\n%#v", err, query)
		return it, errors.Annotatef(err, "query error")
	}
	if _, err = l.conn.Exec(query, iri, time.Now().UTC(), it.GetType(), raw); err != nil {
		l.errFn("query error: %s\n%s\n%#v", err, query)
		return it, errors.Annotatef(err, "query error")
	}
	col, key := path.Split(iri.String())
	if len(key) > 0 && handlers.ValidCollection(handlers.CollectionType(path.Base(col))){
		// Add private items to the collections table
		if colIRI, k := handlers.Split(pub.IRI(col)); k == "" {
			if err := l.AddTo(colIRI, it); err != nil {
				return it, err
			}
		}
	}

	return it, nil
}
