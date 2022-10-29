//go:build storage_pgx || storage_all || (!storage_boltdb && !storage_fs && !storage_badger && !storage_sqlite)

package pgx

import (
	"fmt"
	"path"
	"strings"
	"time"

	"git.sr.ht/~mariusor/lw"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/log"
	"github.com/go-ap/fedbox/storage"
	"github.com/go-ap/jsonld"
	"github.com/go-ap/processing"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/pgtype"
)

var encodeFn = jsonld.Marshal
var decodeFn = jsonld.Unmarshal

type repo struct {
	baseURL string
	conn    *pgx.ConnPool
	conf    Config
	l       lw.Logger
	logFn   loggerFn
	errFn   loggerFn
}

type loggerFn func(lw.Ctx, string, ...interface{})

// IsLocalIRI shows if the received IRI belongs to the current instance
func (r repo) IsLocalIRI(i vocab.IRI) bool {
	return i.Contains(vocab.IRI(r.baseURL), false)
}

func logFn(l lw.Logger, lvl lw.Level) loggerFn {
	return func(w lw.Ctx, f string, par ...interface{}) {
		if l != nil {
			l.WithContext(w).Debugf(f, par...)
		}
	}
}

type Config struct {
	Host     string
	Port     uint16
	Database string
	User     string
	Password string
	BaseURL  string
}

func New(conf Config, url string, lp lw.Logger) (*repo, error) {
	l := repo{
		baseURL: url,
		conf:    conf,
		l:       lp,
		errFn:   logFn(lp, lw.ErrorLevel),
	}

	if err := l.Open(); err != nil {
		return &l, err
	}
	return &l, nil
}

func (r repo) Load(i vocab.IRI) (vocab.Item, error) {
	if !r.IsLocalIRI(vocab.IRI(r.baseURL)) {
		return nil, errors.Newf("unable to load non-local IRI: %s", i)
	}
	f, err := ap.FiltersFromIRI(i)
	if err != nil {
		return nil, err
	}

	return loadOneFromDb(r.conn, getCollectionTable(f.Collection), f)
}

func getCollectionTable(typ vocab.CollectionPath) string {
	switch typ {
	case vocab.Followers:
		fallthrough
	case vocab.Following:
		fallthrough
	case "actors":
		return "actors"
	case vocab.Inbox:
		fallthrough
	case vocab.Outbox:
		fallthrough
	case vocab.Shares:
		fallthrough
	case vocab.Liked:
		fallthrough
	case vocab.Likes:
		fallthrough
	case "activities":
		return "activities"
	case vocab.Replies:
		fallthrough
	default:
		return "objects"
	}
	return "objects"
}

func loadOneFromDb(conn *pgx.ConnPool, table string, f processing.Filterable) (vocab.Item, error) {
	col, _, err := loadFromDb(conn, table, f)
	if err != nil {
		return nil, err
	}
	if vocab.IsNil(col) {
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

func loadFromDb(conn *pgx.ConnPool, table string, f processing.Filterable) (vocab.ItemCollection, uint, error) {
	clauses, values := getWhereClauses(f)
	var total uint = 0

	sel := fmt.Sprintf("SELECT id, key, iri, created_at::timestamptz, type, raw FROM %s WHERE %s ORDER BY raw->>'published' DESC %s", table, strings.Join(clauses, " AND "), getLimit(f))
	rows, err := conn.Query(sel, values...)
	defer rows.Close()
	if err != nil {
		if err == pgx.ErrNoRows {
			return vocab.ItemCollection{}, total, nil
		}
		return nil, total, errors.Annotatef(err, "unable to run select")
	}

	ret := make(vocab.ItemCollection, 0)
	// Iterate through the result set
	for rows.Next() {
		var id int64
		var key ap.Hash
		var iri string
		var created pgtype.Timestamptz
		var typ string
		var raw []byte
		err = rows.Scan(&id, &key, &iri, &created, &typ, &raw)
		if err != nil {
			return ret, total, errors.Annotatef(err, "scan values error")
		}

		it, err := vocab.UnmarshalJSON(raw)
		if err != nil {
			return ret, total, errors.Annotatef(err, "unable to unmarshal raw item")
		}
		ret = append(ret, it)
	}

	selCnt := fmt.Sprintf("SELECT COUNT(id) FROM %s WHERE %s", table, strings.Join(clauses, " AND "))
	if err = conn.QueryRow(selCnt, values...).Scan(&total); err != nil {
		err = errors.Annotatef(err, "unable to count all rows")
	}

	return ret, total, err
}

// Save
func (r repo) Save(it vocab.Item) (vocab.Item, error) {
	if vocab.IsNil(it) {
		return it, errors.Newf("not saving nil item")
	}
	var err error

	var table string
	if vocab.ActivityTypes.Contains(it.GetType()) {
		table = string(ap.ActivitiesType)
	} else if vocab.ActorTypes.Contains(it.GetType()) {
		table = string(ap.ActorsType)
	} else {
		table = string(ap.ObjectsType)
	}

	if len(it.GetLink()) > 0 {
		ff := ap.FiltersNew(
			ap.ItemKey(it.GetLink().String()),
			ap.Type(it.GetType()),
		)
		if _, cnt, _ := loadFromDb(r.conn, table, ff); cnt != 0 {
			err := processing.ErrDuplicateObject("%s in table %s", it.GetLink(), table)
			r.errFn(lw.Ctx{
				"table": table,
				"type":  it.GetType(),
				"iri":   it.GetLink(),
				"err":   err.Error(),
			}, "skipping save")
			return it, err
		}
	}
	it, err = saveToDb(r, table, it)
	if err != nil {
		return it, err
	}

	colIRI := vocab.CollectionPath(table).IRI(vocab.IRI(r.baseURL))
	err = r.AddTo(colIRI, it)
	if err != nil {
		// This errs
		r.errFn(lw.Ctx{"IRI": it.GetLink(), "collection": colIRI}, "unable to add to collection")
	}

	// TODO(marius) Move to somewhere else
	if toFw, ok := it.(vocab.HasRecipients); ok {
		for _, fw := range toFw.Recipients() {
			colIRI := fw.GetLink()
			if r.IsLocalIRI(colIRI) {
				// we shadow the err variable intentionally so it does not propagate upper to the call stack
				r.AddTo(colIRI, it)
			}
		}
	}

	return it, err
}

// Create
func (r repo) Create(it vocab.CollectionInterface) (vocab.CollectionInterface, error) {
	if vocab.IsNil(it) {
		return it, errors.Newf("unable to create nil collection")
	}
	if len(it.GetLink()) == 0 {
		return it, errors.Newf("Invalid create collection does not have a valid IRI")
	}

	query := fmt.Sprintf("INSERT INTO collections (iri, type, created_at) VALUES ($1, $2, $3::timestamptz);")

	now := time.Now().UTC()
	nowTz := pgtype.Timestamptz{
		Time:   now,
		Status: pgtype.Present,
	}
	_, err := r.conn.Exec(query, it.GetLink(), it.GetType(), &nowTz)
	if err != nil {
		r.errFn(lw.Ctx{
			"err": err.Error(),
		}, "query error")
		return it, errors.Annotatef(err, "query error")
	}

	return it, nil
}

// RemoveFrom
func (r repo) RemoveFrom(col vocab.IRI, it vocab.Item) error {
	return errors.NotImplementedf("removing from collection is not yet implemented")
}

// AddTo
func (r repo) AddTo(col vocab.IRI, it vocab.Item) error {
	if vocab.IsNil(it) {
		return errors.Newf("unable to add nil element to collection")
	}
	if len(col) == 0 {
		return errors.Newf("unable to find collection")
	}
	if len(it.GetLink()) == 0 {
		return errors.Newf("Invalid create collection does not have a valid IRI")
	}
	if !r.IsLocalIRI(col.GetLink()) {
		return errors.Newf("Unable to save to non local collection %s", col)
	}

	query := fmt.Sprintf("UPDATE collections SET updated_at = $1, elements = array_append(elements, $2), count = count+1 WHERE iri = $3;")

	now := time.Now().UTC()
	nowTz := pgtype.Timestamptz{
		Time:   now,
		Status: pgtype.Present,
	}
	t, err := r.conn.Exec(query, &nowTz, it.GetLink(), col)
	if err != nil {
		r.errFn(lw.Ctx{
			"err": err.Error(),
		}, "query error")
		return errors.Annotatef(err, "query error")
	}
	if t.RowsAffected() != 1 {
		r.errFn(lw.Ctx{
			"rows": t.RowsAffected(),
		}, "query error")
		return errors.Annotatef(err, "query error, Invalid updated rows")
	}

	return nil
}

func saveToDb(l repo, table string, it vocab.Item) (vocab.Item, error) {
	query := fmt.Sprintf("INSERT INTO %s (key, iri, created_at, type, raw) VALUES ($1, $2, $3::timestamptz, $4, $5::jsonb);", table)

	iri := it.GetLink()
	uuid := path.Base(iri.String())
	if uuid == "." {
		// broken ID generation
		return it, errors.Newf("Unable to generate ID for %s[%s]", table, it.GetType())
	}
	raw, _ := encodeFn(it)
	nowTz := pgtype.Timestamptz{
		Time:   time.Now().UTC(),
		Status: pgtype.Present,
	}
	_, err := l.conn.Exec(query, uuid, iri, &nowTz, it.GetType(), raw)
	if err != nil {
		l.errFn(lw.Ctx{
			"err": err.Error(),
		}, "query error")
		return it, errors.Annotatef(err, "query error")
	}

	return it, nil
}

func (r repo) deleteItem(table string, it vocab.Item) error {
	iri := it.GetLink()
	if len(iri) == 0 {
		return errors.Newf("Invalid update item does not have a valid IRI")
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE iri = $1;", table)
	if _, err := r.conn.Exec(query, iri); err != nil {
		r.errFn(lw.Ctx{
			"err": err.Error(),
		}, "query error")
		return errors.Annotatef(err, "query error")
	}

	return nil
}

func (r repo) updateItem(table string, it vocab.Item) (vocab.Item, error) {
	if table == "activities" {
		return it, errors.Newf("update action Invalid, activities are immutable")
	}
	iri := it.GetLink()
	if len(iri) == 0 {
		return it, errors.Newf("Invalid update item does not have a valid IRI")
	}

	query := fmt.Sprintf("UPDATE %s SET type = $1, updated_at = $2::timestamptz,raw = $3::jsonb WHERE iri = $4;", table)
	now := time.Now().UTC()
	if vocab.ActorTypes.Contains(it.GetType()) {
		if p, err := vocab.ToActor(it); err == nil {
			p.Updated = now
			it = p
		}
	} else if vocab.ObjectTypes.Contains(it.GetType()) && it.GetType() != vocab.TombstoneType {
		if o, err := vocab.ToObject(it); err == nil {
			o.Updated = now
			it = o
		}
	}
	raw, _ := encodeFn(it)
	nowTz := pgtype.Timestamptz{Time: now, Status: pgtype.Present}

	if _, err := r.conn.Exec(query, it.GetType(), &nowTz, raw, iri); err != nil {
		r.errFn(lw.Ctx{
			"err": err.Error(),
		}, "query error")
		return it, errors.Annotatef(err, "query error")
	}

	return it, nil
}

// Delete
func (r repo) Delete(it vocab.Item) error {
	if vocab.IsNil(it) {
		return errors.Newf("not saving nil item")
	}
	var table string

	if vocab.ActivityTypes.Contains(it.GetType()) {
		return errors.Newf("unable to delete activity")
	} else if vocab.ActorTypes.Contains(it.GetType()) {
		table = "actors"
	} else {
		table = "objects"
	}

	f := ap.FiltersNew()
	f.IRI = it.GetLink()
	if it.IsObject() {
		f.Type = ap.CompStrs{ap.StringEquals(string(it.GetType()))}
	}
	var cnt uint
	var found vocab.ItemCollection
	found, cnt, _ = loadFromDb(r.conn, table, f)
	if cnt == 0 {
		if table == "objects" {
			table = "actors"
		}
		// try other table
		found, cnt, _ = loadFromDb(r.conn, table, f)
	}
	if cnt == 0 {
		err := errors.NotFoundf("%s in either actors or objects", it.GetLink())
		r.errFn(lw.Ctx{
			"table": table,
			"type":  it.GetType(),
			"iri":   it.GetLink(),
			"err":   err.Error(),
		}, "unable to find old item")
		return err
	}
	old := found.First()

	t := vocab.Tombstone{
		ID:   vocab.ID(it.GetLink()),
		Type: vocab.TombstoneType,
		To: vocab.ItemCollection{
			vocab.PublicNS,
		},
		Deleted:    time.Now().UTC(),
		FormerType: old.GetType(),
	}

	return r.deleteItem(table, t)
}

// Open opens the underlying db connections
func (r *repo) Open() error {
	var err error
	r.conn, err = pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig: pgx.ConnConfig{
			Host:     r.conf.Host,
			Port:     r.conf.Port,
			Database: r.conf.Database,
			User:     r.conf.User,
			Password: r.conf.Password,
			Logger:   log.NewPgxLogger(r.l),
		},
		MaxConnections: 3,
	})
	if err != nil {
		return err
	}
	return nil
}

// Close closes the underlying db connections
func (r *repo) Close() error {
	r.conn.Close()
	return nil
}

// PasswordSet
func (r *repo) PasswordSet(it vocab.Item, pw []byte) error {
	return errors.NotImplementedf("PasswordSet is not implemented by the postgres storage layer")
}

func (r *repo) PasswordCheck(it vocab.Item, pw []byte) error {
	return errors.NotImplementedf("PasswordCheck is not implemented by the postgres storage layer")
}

// LoadMetadata
func (r *repo) LoadMetadata(iri vocab.IRI) (*storage.Metadata, error) {
	return nil, errors.NotImplementedf("LoadMetadata is not implemented by the postgres storage layer")
}

// SaveMetadata
func (r *repo) SaveMetadata(m storage.Metadata, iri vocab.IRI) error {
	return errors.NotImplementedf("SaveMetadata is not implemented by the postgres storage layer")
}
