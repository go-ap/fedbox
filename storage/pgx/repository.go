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
	uuid2 "github.com/google/uuid"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/pgtype"
	"github.com/sirupsen/logrus"
	"net/url"
	"path"
	"strings"
	"time"
)

type loader struct {
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
func (l loader) IsLocalIRI(i as.IRI) bool {
	if _, err := url.Parse(i.String()); err != nil {
		// not an url
		l.errFn(logrus.Fields{
			"IRI": i,
		}, "Invalid url")
		return false
	}
	return strings.Contains(i.String(), l.baseURL)
}

func logFn(l logrus.FieldLogger, lvl logrus.Level) loggerFn {
	return func(w logrus.Fields, f string, par ...interface{}) {
		if l != nil {
			l.WithFields(w).Logf(lvl, f, par...)
		}
	}
}

func New(conf config.BackendConfig, url string, lp logrus.FieldLogger) (*loader, error) {
	l := loader{
		baseURL: url,
		conf:    conf,
		d:       client.NewClient(),
		l: lp,
		errFn:   logFn(lp, logrus.ErrorLevel),
	}

	if err := l.Open(); err != nil {
		return &l, err
	}
	return &l, nil
}

func (l loader) LoadActivities(ff s.Filterable) (as.ItemCollection, uint, error) {
	f, ok := ff.(*ap.Filters)
	if !ok {
		return nil, 0, errors.Newf("Invalid ActivityPub filters")
	}
	return loadFromDb(l.conn, "activities", f)
}

func (l loader) LoadActors(ff s.Filterable) (as.ItemCollection, uint, error) {
	f, ok := ff.(*ap.Filters)
	if !ok {
		return nil, 0, errors.Newf("Invalid ActivityPub filters")
	}
	return loadFromDb(l.conn, "actors", f)
}

func (l loader) LoadObjects(ff s.Filterable) (as.ItemCollection, uint, error) {
	f, ok := ff.(*ap.Filters)
	if !ok {
		return nil, 0, errors.Newf("Invalid ActivityPub filters")
	}
	return loadFromDb(l.conn, "objects", f)
}

func GetCollectionTable(typ handlers.CollectionType) string {
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

func (l loader) LoadCollection(ff s.Filterable) (as.CollectionInterface, error) {
	id := ff.ID()
	colFilters := ap.Filters{
		IRI: id,
	}
	clauses, values := colFilters.GetWhereClauses()

	var ret as.CollectionInterface
	sel := fmt.Sprintf("SELECT id, iri, created_at::timestamptz, type, count, elements FROM collections WHERE %s ORDER BY created_at DESC LIMIT 1", strings.Join(clauses, " AND "))
	rows, err := l.conn.Query(sel, values...)
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
		l.l.WithFields(logrus.Fields{
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
		items, total, err = loadFromDb(l.conn, GetCollectionTable(f.Collection), f)
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

func (l loader) Load(ff s.Filterable) (as.ItemCollection, uint, error) {
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

func (l loader) SaveActivity(it as.Item) (as.Item, error) {
	var err error

	it = ap.FlattenProperties(it)
	it, err = l.SaveObject(it)
	if err != nil {
		l.errFn(logrus.Fields{"IRI": it.GetLink()}, "unable to save activity")
		return it, err
	}

	return it, err
}

func getCollectionID(actor as.Item, c handlers.CollectionType) as.ObjectID {
	return as.ObjectID(fmt.Sprintf("%s/%s", actor.GetLink(), c))
}

func getCollectionIRI(actor as.Item, c handlers.CollectionType) as.IRI {
	return as.IRI(fmt.Sprintf("%s/%s", actor.GetLink(), c))
}

func (l loader) createActorCollection(actor as.Item, c handlers.CollectionType) (as.CollectionInterface, error) {
	col := as.OrderedCollection{
		Parent: as.Parent{
			ID:   getCollectionID(actor, c),
			Type: as.OrderedCollectionType,
		},
	}
	return createCollection(l, &col)
}

func (l loader) createObjectCollection(object as.Item, c handlers.CollectionType) (as.CollectionInterface, error) {
	col := as.OrderedCollection{
		Parent: as.Parent{
			ID:   getCollectionID(object, c),
			Type: as.OrderedCollectionType,
		},
	}
	return createCollection(l, &col)
}

func (l loader) SaveActor(it as.Item) (as.Item, error) {
	return l.SaveObject(it)
}

func (l loader) SaveObject(it as.Item) (as.Item, error) {
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
		if _, cnt, _ := loadFromDb(l.conn, table, &ap.Filters{
			ItemKey: []ap.Hash{ap.Hash(it.GetLink().String())},
			Type:    []as.ActivityVocabularyType{it.GetType()},
		}); cnt != 0 {
			err := ap.ErrDuplicateObject("%s in table %s", it.GetLink(), table)
			l.errFn(logrus.Fields{
				"table": table,
				"type":  it.GetType(),
				"iri":   it.GetLink(),
				"err":   err.Error(),
			}, "skipping save")
			return it, err
		}
	}
	it, err = saveToDb(l, table, it)
	if err != nil {
		return it, err
	}

	colIRI := getCollectionIRI(as.IRI(l.baseURL), handlers.CollectionType(table))
	err = addToCollection(l, colIRI, it)
	if err != nil {
		// This errs
		l.errFn(logrus.Fields{"IRI": it.GetLink(), "collection": colIRI}, "unable to add to collection")
	}

	// TODO(marius) Move to somewhere else
	if toFw, ok := it.(as.HasRecipients); ok {
		for _, fw := range toFw.Recipients() {
			colIRI := fw.GetLink()
			if l.IsLocalIRI(colIRI) {
				// we shadow the err variable intentionally so it does not propagate upper to the call stack
				if errFw := addToCollection(l, colIRI, it); err != nil {
					l.errFn(logrus.Fields{"IRI": it.GetLink(), "collection": colIRI, "error": errFw}, "unable to add to collection")
				}
			}
		}
	}

	return it, err
}

func createCollection(l loader, it as.CollectionInterface) (as.CollectionInterface, error) {
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
	_, err := l.conn.Exec(query, it.GetLink(), it.GetType(), &nowTz)
	if err != nil {
		l.errFn(logrus.Fields{
			"err": err.Error(),
		}, "query error")
		return it, errors.Annotatef(err, "query error")
	}

	return it, nil
}

func addToCollection(l loader, iri as.IRI, it as.Item) error {
	if it == nil {
		return errors.Newf("unable to add nil element to collection")
	}
	if len(iri) == 0 {
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
	t, err := l.conn.Exec(query, &nowTz, it.GetLink(), iri)
	if err != nil {
		l.errFn(logrus.Fields{
			"err": err.Error(),
		}, "query error")
		return errors.Annotatef(err, "query error")
	}
	if t.RowsAffected() != 1 {
		l.errFn(logrus.Fields{
			"rows": t.RowsAffected(),
		}, "query error")
		return errors.Annotatef(err, "query error, Invalid updated rows")
	}

	return nil
}

func saveToDb(l loader, table string, it as.Item) (as.Item, error) {
	query := fmt.Sprintf("INSERT INTO %s (key, iri, created_at, type, raw) VALUES ($1, $2, $3::timestamptz, $4, $5::jsonb);", table)

	iri := it.GetLink()
	if len(iri) == 0 {
		// TODO(marius): this needs to be in a different place

		pc := as.IRI(fmt.Sprintf("%s/%s", l.baseURL, table))

		if _, err := l.GenerateID(it, pc, nil); err != nil {
			return it, err
		}
		if as.ActorTypes.Contains(it.GetType()) {
			if p, err := ap.ToPerson(it); err == nil {
				if in, err := l.createActorCollection(it, handlers.Inbox); err != nil {
					return it, err
				} else {
					p.Inbox = in.GetLink()
				}
				if out, err := l.createActorCollection(it, handlers.Outbox); err != nil {
					return it, err
				} else {
					p.Outbox = out.GetLink()
				}
				if fers, err := l.createActorCollection(it, handlers.Followers); err != nil {
					return it, err
				} else {
					p.Followers = fers.GetLink()
				}
				if fing, err := l.createActorCollection(it, handlers.Following); err != nil {
					return it, err
				} else {
					p.Following = fing.GetLink()
				}
				if ld, err := l.createActorCollection(it, handlers.Liked); err != nil {
					return it, err
				} else {
					p.Liked = ld.GetLink()
				}
				if ls, err := l.createActorCollection(it, handlers.Likes); err != nil {
					return it, err
				} else {
					p.Likes = ls.GetLink()
				}
				if sh, err := l.createActorCollection(it, handlers.Shares); err != nil {
					return it, err
				} else {
					p.Shares = sh.GetLink()
				}
				it = p
			}
		} else if as.ObjectTypes.Contains(it.GetType()) {
			if o, err := as.ToObject(it); err == nil {
				if repl, err := l.createObjectCollection(it, handlers.Replies); err != nil {
					return it, err
				} else {
					o.Replies = repl.GetLink()
				}
				it = o
			}
		}
	}

	iri = it.GetLink()
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

func (l loader) updateItem(table string, it as.Item) (as.Item, error) {
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
	_, err := l.conn.Exec(query, it.GetType(), &nowTz, raw, iri)
	if err != nil {
		l.errFn(logrus.Fields{
			"err": err.Error(),
		}, "query error")
		return it, errors.Annotatef(err, "query error")
	}

	return it, nil
}

func (l loader) UpdateActor(it as.Item) (as.Item, error) {
	return l.UpdateObject(it)
}

func (l loader) UpdateObject(it as.Item) (as.Item, error) {
	if it == nil {
		return it, errors.Newf("not saving nil item")
	}
	var err error
	var table string
	var found as.ItemCollection
	var cnt uint
	label := "item"
	if as.ActivityTypes.Contains(it.GetType()) {
		return nil, errors.Newf("unable to update activity")
	} else if as.ActorTypes.Contains(it.GetType()) {
		label = "actor"
		table = "actors"
		found, cnt, _ = l.LoadActors(&ap.Filters{
			ItemKey: []ap.Hash{ap.Hash(it.GetLink().String())},
			Type:    []as.ActivityVocabularyType{it.GetType()},
		})
	} else {
		label = "object"
		table = "objects"
		found, cnt, _ = l.LoadObjects(&ap.Filters{
			ItemKey: []ap.Hash{ap.Hash(it.GetLink().String())},
			Type:    []as.ActivityVocabularyType{it.GetType()},
		})
	}
	if len(it.GetLink()) == 0 {
		err := errors.NotFoundf("Unable to update %s with no ID", label)
		return it, err
	}

	if cnt == 0 {
		err := errors.NotFoundf("%s %s", it.GetLink(), label)
		l.errFn(logrus.Fields{
			"type": it.GetType(),
			"iri":  it.GetLink(),
			"err":  err.Error(),
		}, "unable to find old item")
		return it, err
	}
	old := found[0]
	it, err = ap.UpdateItemProperties(old, it)
	if err != nil {
		l.errFn(logrus.Fields{
			"table": table,
			"type":  old.GetType(),
			"iri":   old.GetLink(),
			"err":   err.Error(),
		}, "Invalid")
	}

	it, err = l.updateItem(table, it)
	if err != nil {
		l.errFn(logrus.Fields{
			"action": "update",
			"table":  table,
		}, "%s", err.Error())
	}

	return it, err
}

func (l loader) DeleteActor(it as.Item) (as.Item, error) {
	return l.DeleteObject(it)
}

// GenerateID generates an unique identifier for the it ActivityPub Object.
// TODO(marius): remove the need to
func (l loader) GenerateID(it as.Item, partOf as.IRI, by as.Item) (as.ObjectID, error) {
	id := as.ObjectID(fmt.Sprintf("%s/%s", strings.ToLower(string(partOf)), uuid2.New()))

	if as.ActivityTypes.Contains(it.GetType()) {
		a, err := ap.ToActivity(it)
		if err != nil {
			return *it.GetID(), err
		}
		a.ID = id
		it = a
	}
	if as.ActorTypes.Contains(it.GetType()) {
		p, err := ap.ToPerson(it)
		if err != nil {
			return *it.GetID(), err
		}
		p.ID = id
		it = p
	}
	if as.ObjectTypes.Contains(it.GetType()) {
		switch it.GetType() {
		case as.PlaceType:
			p, err := as.ToPlace(it)
			if err != nil {
				return *it.GetID(), err
			}
			p.ID = id
			it = p
		case as.ProfileType:
			p, err := as.ToProfile(it)
			if err != nil {
				return *it.GetID(), err
			}
			p.ID = id
			it = p
		case as.RelationshipType:
			r, err := as.ToRelationship(it)
			if err != nil {
				return *it.GetID(), err
			}
			r.ID = id
			it = r
		case as.TombstoneType:
			p, err := as.ToTombstone(it)
			if err != nil {
				return *it.GetID(), err
			}
			p.ID = id
			it = p
		default:
			o, err := as.ToObject(it)
			if err != nil {
				return *it.GetID(), err
			}
			o.ID = id
			it = o
		}
	}
	return *it.GetID(), nil
}

func (l loader) DeleteObject(it as.Item) (as.Item, error) {
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
		ItemKey: []ap.Hash{ap.Hash(it.GetLink().String())},
	}
	if it.IsObject() {
		f.Type = []as.ActivityVocabularyType{it.GetType()}
	}
	var cnt uint
	var found as.ItemCollection
	found, cnt, _ = loadFromDb(l.conn, table, &f)
	if cnt == 0 {
		if table == "objects" {
			table = "actors"
		}
		// try other table
		found, cnt, _ = loadFromDb(l.conn, table, &f)
	}
	if cnt == 0 {
		err := errors.NotFoundf("%s in either actors or objects", it.GetLink())
		l.errFn(logrus.Fields{
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
				as.IRI(ap.Public),
			},
		},
		Deleted:    time.Now().UTC(),
		FormerType: old.GetType(),
	}

	return l.updateItem(table, t)
}

// Close closes the underlying db connections
func (l *loader) Open() error {
	var err error
	l.conn, err = pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig: pgx.ConnConfig {
			Host:     l.conf.Host,
			Port:     uint16(l.conf.Port),
			Database: l.conf.Name,
			User:     l.conf.User,
			Password: l.conf.Pw,
			Logger:   DBLogger(l.l),
		},
		MaxConnections: 3,
	})
	if err != nil {
		return err
	}
	return nil
}

// Close closes the underlying db connections
func (l *loader) Close() error {
	l.conn.Close()
	return nil
}
