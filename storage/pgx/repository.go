package pgx

import (
	"fmt"
	"github.com/go-ap/activitypub/client"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/handlers"
	"github.com/go-ap/jsonld"
	s "github.com/go-ap/storage"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/pgtype"
	"github.com/sirupsen/logrus"
	"net/url"
	"path"
	"strings"
	"time"
)

type repo struct {
	baseURL string
	conn    *pgx.ConnPool
	conf    config.BackendConfig
	d       client.Client
	l       logrus.FieldLogger
	logFn   loggerFn
	errFn   loggerFn
}

type loggerFn func(logrus.Fields, string, ...interface{})

// IsLocalIRI shows if the received IRI belongs to the current instance
func (r repo) IsLocalIRI(i as.IRI) bool {
	if _, err := url.Parse(i.String()); err != nil {
		// not an url
		r.errFn(logrus.Fields{
			"IRI": i,
		}, "Invalid url")
		return false
	}
	return strings.Contains(i.String(), r.baseURL)
}

func logFn(l logrus.FieldLogger, lvl logrus.Level) loggerFn {
	return func(w logrus.Fields, f string, par ...interface{}) {
		if l != nil {
			l.WithFields(w).Logf(lvl, f, par...)
		}
	}
}

func New(conf config.BackendConfig, url string, lp logrus.FieldLogger) (*repo, error) {
	l := repo{
		baseURL: url,
		conf:    conf,
		d:       client.NewClient(),
		l:       lp,
		errFn:   logFn(lp, logrus.ErrorLevel),
	}

	if err := l.Open(); err != nil {
		return &l, err
	}
	return &l, nil
}

func (r repo) LoadActivities(ff s.Filterable) (as.ItemCollection, uint, error) {
	f, ok := ff.(*ap.Filters)
	if !ok {
		return nil, 0, errors.Newf("Invalid ActivityPub filters")
	}
	return loadFromDb(r.conn, "activities", f)
}

func (r repo) LoadActors(ff s.Filterable) (as.ItemCollection, uint, error) {
	f, ok := ff.(*ap.Filters)
	if !ok {
		return nil, 0, errors.Newf("Invalid ActivityPub filters")
	}
	return loadFromDb(r.conn, "actors", f)
}

func (r repo) LoadObjects(ff s.Filterable) (as.ItemCollection, uint, error) {
	f, ok := ff.(*ap.Filters)
	if !ok {
		return nil, 0, errors.Newf("Invalid ActivityPub filters")
	}
	return loadFromDb(r.conn, "objects", f)
}

func getCollectionTable(typ handlers.CollectionType) string {
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

func (r repo) LoadCollection(ff s.Filterable) (as.CollectionInterface, error) {
	id := ff.ID()
	colFilters := ap.Filters{
		IRI: id,
	}
	clauses, values := colFilters.GetWhereClauses()

	var ret as.CollectionInterface
	sel := fmt.Sprintf("SELECT id, iri, created_at::timestamptz, type, count, elements FROM collections WHERE %s ORDER BY created_at DESC LIMIT 1", strings.Join(clauses, " AND "))
	rows, err := r.conn.Query(sel, values...)
	defer rows.Close()
	if err != nil {
		if err == pgx.ErrNoRows {
			return ret, nil
		}
		return ret, errors.Annotatef(err, "unable to run select")
	}
	if err := rows.Err(); err != nil {
		return ret, errors.Annotatef(err, "unable to run select")
	}

	f, ok := ff.(*ap.Filters)
	if !ok {
		return ret, errors.Newf("unable to load filters")
	}
	var count int
	// Iterate through the result set
	for rows.Next() {
		var id int64
		var iri string
		var created pgtype.Timestamptz
		var typ string
		var elements []string
		err = rows.Scan(&id, &iri, &created, &typ, &count, &elements)
		if err != nil {
			return ret, errors.Annotatef(err, "scan values error")
		}

		if as.ActivityVocabularyType(typ) == as.CollectionType {
			col := &as.Collection{}
			col.ID = as.ObjectID(iri)
			col.Type = as.CollectionType
			col.TotalItems = uint(count)
			ret = col
		}
		if as.ActivityVocabularyType(typ) == as.OrderedCollectionType {
			col := &as.OrderedCollection{}
			col.ID = as.ObjectID(iri)
			col.Type = as.OrderedCollectionType
			col.TotalItems = uint(count)
			ret = col
		}
		if count == 0 {
			return ret, nil
		}
		r.l.WithFields(logrus.Fields{
			"id":         id,
			"iri":        iri,
			"created_at": created,
			"type":       typ,
			"count":      count,
			"elements":   elements,
		}).Infof("loaded fields")

		var items as.ItemCollection
		f.ItemKey = f.ItemKey[:0]
		f.IRI = ""
		for _, elem := range elements {
			f.ItemKey = append(f.ItemKey, ap.Hash(elem))
		}
		var total uint
		items, total, err = loadFromDb(r.conn, getCollectionTable(f.Collection), f)
		if as.ActivityVocabularyType(typ) == as.CollectionType {
			if col, err := ap.ToCollection(ret); err == nil {
				col.TotalItems = total
			}
		}
		if as.ActivityVocabularyType(typ) == as.OrderedCollectionType {
			if col, err := ap.ToOrderedCollection(ret); err == nil {
				col.TotalItems = total
			}
		}

		if err == nil && total > 0 {
			for _, it := range items {
				ret.Append(it)
			}
		}
	}
	if ret == nil {
		return ret, errors.Newf("could not load '%s' collection", f.Collection)
	}
	return ret, err
}

func (r repo) Load(ff s.Filterable) (as.ItemCollection, uint, error) {
	return nil, 0, errors.NotImplementedf("not implemented loader.Load()")
}

func loadFromDb(conn *pgx.ConnPool, table string, f ap.Filterable) (as.ItemCollection, uint, error) {
	clauses, values := f.GetWhereClauses()
	var total uint = 0

	sel := fmt.Sprintf("SELECT id, key, iri, created_at::timestamptz, type, raw FROM %s WHERE %s ORDER BY raw->>'published' DESC %s", table, strings.Join(clauses, " AND "), f.GetLimit())
	rows, err := conn.Query(sel, values...)
	defer rows.Close()
	if err != nil {
		if err == pgx.ErrNoRows {
			return as.ItemCollection{}, total, nil
		}
		return nil, total, errors.Annotatef(err, "unable to run select")
	}

	ret := make(as.ItemCollection, 0)
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

		it, err := as.UnmarshalJSON(raw)
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

func (r repo) SaveActivity(it as.Item) (as.Item, error) {
	var err error

	it, err = r.SaveObject(it)
	if err != nil {
		r.errFn(logrus.Fields{"IRI": it.GetLink()}, "unable to save activity")
		return it, err
	}

	return it, err
}

func getCollectionIRI(actor as.Item, c handlers.CollectionType) as.IRI {
	return as.IRI(fmt.Sprintf("%s/%s", actor.GetLink(), c))
}

func (r repo) SaveActor(it as.Item) (as.Item, error) {
	return r.SaveObject(it)
}

func (r repo) SaveObject(it as.Item) (as.Item, error) {
	if it == nil {
		return it, errors.Newf("not saving nil item")
	}
	var err error

	var table string
	if as.ActivityTypes.Contains(it.GetType()) {
		table = "activities"
	} else if as.ActorTypes.Contains(it.GetType()) {
		table = "actors"
	} else {
		table = "objects"
	}

	if len(it.GetLink()) > 0 {
		if _, cnt, _ := loadFromDb(r.conn, table, &ap.Filters{
			ItemKey: []ap.Hash{ap.Hash(it.GetLink().String())},
			Type:    []as.ActivityVocabularyType{it.GetType()},
		}); cnt != 0 {
			err := ap.ErrDuplicateObject("%s in table %s", it.GetLink(), table)
			r.errFn(logrus.Fields{
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

	colIRI := getCollectionIRI(as.IRI(r.baseURL), handlers.CollectionType(table))
	err = r.AddToCollection(colIRI, it)
	if err != nil {
		// This errs
		r.errFn(logrus.Fields{"IRI": it.GetLink(), "collection": colIRI}, "unable to add to collection")
	}

	// TODO(marius) Move to somewhere else
	if toFw, ok := it.(as.HasRecipients); ok {
		for _, fw := range toFw.Recipients() {
			colIRI := fw.GetLink()
			if r.IsLocalIRI(colIRI) {
				// we shadow the err variable intentionally so it does not propagate upper to the call stack
				if errFw := r.AddToCollection(colIRI, it); err != nil {
					r.errFn(logrus.Fields{"IRI": it.GetLink(), "collection": colIRI, "error": errFw}, "unable to add to collection")
				}
			}
		}
	}

	return it, err
}

func (r repo) CreateCollection(it as.CollectionInterface) (as.CollectionInterface, error) {
	if it == nil {
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
		r.errFn(logrus.Fields{
			"err": err.Error(),
		}, "query error")
		return it, errors.Annotatef(err, "query error")
	}

	return it, nil
}

func (r repo) AddToCollection(col as.IRI, it as.Item) error {
	if it == nil {
		return errors.Newf("unable to add nil element to collection")
	}
	if len(col) == 0 {
		return errors.Newf("unable to find collection")
	}
	if len(it.GetLink()) == 0 {
		return errors.Newf("Invalid create collection does not have a valid IRI")
	}

	query := fmt.Sprintf("UPDATE collections SET updated_at = $1, elements = array_append(elements, $2), count = count+1 WHERE iri = $3;")

	now := time.Now().UTC()
	nowTz := pgtype.Timestamptz{
		Time:   now,
		Status: pgtype.Present,
	}
	t, err := r.conn.Exec(query, &nowTz, it.GetLink(), col)
	if err != nil {
		r.errFn(logrus.Fields{
			"err": err.Error(),
		}, "query error")
		return errors.Annotatef(err, "query error")
	}
	if t.RowsAffected() != 1 {
		r.errFn(logrus.Fields{
			"rows": t.RowsAffected(),
		}, "query error")
		return errors.Annotatef(err, "query error, Invalid updated rows")
	}

	return nil
}

func saveToDb(l repo, table string, it as.Item) (as.Item, error) {
	query := fmt.Sprintf("INSERT INTO %s (key, iri, created_at, type, raw) VALUES ($1, $2, $3::timestamptz, $4, $5::jsonb);", table)

	iri := it.GetLink()
	uuid := path.Base(iri.String())
	if uuid == "." {
		// broken ObjectID generation
		return it, errors.Newf("Unable to generate ObjectID for %s[%s]", table, it.GetType())
	}
	raw, _ := jsonld.Marshal(it)
	nowTz := pgtype.Timestamptz{
		Time:   time.Now().UTC(),
		Status: pgtype.Present,
	}
	_, err := l.conn.Exec(query, uuid, iri, &nowTz, it.GetType(), raw)
	if err != nil {
		l.errFn(logrus.Fields{
			"err": err.Error(),
		}, "query error")
		return it, errors.Annotatef(err, "query error")
	}

	return it, nil
}

func (r repo) updateItem(table string, it as.Item) (as.Item, error) {
	if table == "activities" {
		return it, errors.Newf("update action Invalid, activities are immutable")
	}
	iri := it.GetLink()
	if len(iri) == 0 {
		return it, errors.Newf("Invalid update item does not have a valid IRI")
	}

	query := fmt.Sprintf("UPDATE %s SET type = $1, updated_at = $2::timestamptz,raw = $3::jsonb WHERE iri = $4;", table)
	now := time.Now().UTC()
	if as.ActorTypes.Contains(it.GetType()) {
		if p, err := ap.ToPerson(it); err == nil {
			p.Updated = now
			it = p
		}
	} else if as.ObjectTypes.Contains(it.GetType()) && it.GetType() != as.TombstoneType {
		if o, err := as.ToObject(it); err == nil {
			o.Updated = now
			it = o
		}
	}
	raw, _ := jsonld.Marshal(it)

	nowTz := pgtype.Timestamptz{
		Time:   now,
		Status: pgtype.Present,
	}
	_, err := r.conn.Exec(query, it.GetType(), &nowTz, raw, iri)
	if err != nil {
		r.errFn(logrus.Fields{
			"err": err.Error(),
		}, "query error")
		return it, errors.Annotatef(err, "query error")
	}

	return it, nil
}

func (r repo) UpdateActor(it as.Item) (as.Item, error) {
	return r.UpdateObject(it)
}

func (r repo) UpdateObject(it as.Item) (as.Item, error) {
	if it == nil {
		return it, errors.Newf("not saving nil item")
	}
	var err error
	var table string
	label := "item"
	if as.ActivityTypes.Contains(it.GetType()) {
		return nil, errors.Newf("unable to update activity")
	} else if as.ActorTypes.Contains(it.GetType()) {
		label = "actor"
		table = "actors"
	} else {
		label = "object"
		table = "objects"
	}
	if len(it.GetLink()) == 0 {
		err := errors.NotFoundf("Unable to update %s with no ID", label)
		return it, err
	}

	it, err = r.updateItem(table, it)
	if err != nil {
		r.errFn(logrus.Fields{
			"action": "update",
			"table":  table,
		}, "%s", err.Error())
	}

	return it, err
}

func (r repo) DeleteActor(it as.Item) (as.Item, error) {
	return r.DeleteObject(it)
}

// GenerateID generates an unique identifier for the it ActivityPub Object.
func (r repo) GenerateID(it as.Item, by as.Item) (as.ObjectID, error) {
	typ := it.GetType()
	var partOf string
	if as.ActivityTypes.Contains(typ) {
		partOf = fmt.Sprintf("%s/activities", r.baseURL)
	} else if as.ActorTypes.Contains(typ) {
		partOf = fmt.Sprintf("%s/actors", r.baseURL)
	} else if as.ObjectTypes.Contains(typ) {
		partOf = fmt.Sprintf("%s/objects", r.baseURL)
	}
	return ap.GenerateID(it, partOf, by)
}

func (r repo) DeleteObject(it as.Item) (as.Item, error) {
	if it == nil {
		return it, errors.Newf("not saving nil item")
	}
	var table string

	if as.ActivityTypes.Contains(it.GetType()) {
		return nil, errors.Newf("unable to delete activity")
	} else if as.ActorTypes.Contains(it.GetType()) {
		table = "actors"
	} else {
		table = "objects"
	}

	f := ap.Filters{
		IRI: it.GetLink(),
	}
	if it.IsObject() {
		f.Type = []as.ActivityVocabularyType{it.GetType()}
	}
	var cnt uint
	var found as.ItemCollection
	found, cnt, _ = loadFromDb(r.conn, table, &f)
	if cnt == 0 {
		if table == "objects" {
			table = "actors"
		}
		// try other table
		found, cnt, _ = loadFromDb(r.conn, table, &f)
	}
	if cnt == 0 {
		err := errors.NotFoundf("%s in either actors or objects", it.GetLink())
		r.errFn(logrus.Fields{
			"table": table,
			"type":  it.GetType(),
			"iri":   it.GetLink(),
			"err":   err.Error(),
		}, "unable to find old item")
		return it, err
	}
	old := found.First()

	t := as.Tombstone{
		Parent: as.Parent{
			ID:   as.ObjectID(it.GetLink()),
			Type: as.TombstoneType,
			To: as.ItemCollection{
				ap.ActivityStreamsPublicNS,
			},
		},
		Deleted:    time.Now().UTC(),
		FormerType: old.GetType(),
	}

	return r.updateItem(table, t)
}

// Close closes the underlying db connections
func (r *repo) Open() error {
	var err error
	r.conn, err = pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig: pgx.ConnConfig{
			Host:     r.conf.Host,
			Port:     uint16(r.conf.Port),
			Database: r.conf.Name,
			User:     r.conf.User,
			Password: r.conf.Pw,
			Logger:   DBLogger(r.l),
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
