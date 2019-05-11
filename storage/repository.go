package storage

import (
	"fmt"
	"github.com/go-ap/activitypub/client"
	s "github.com/go-ap/activitypub/storage"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/errors"
	"github.com/go-ap/jsonld"
	uuid2 "github.com/google/uuid"
	"github.com/jackc/pgx"
	"github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
	"strings"
)

type Paginator = s.Paginator

type errDuplicateKey struct {
	errors.Err
}

func IsDuplicateKey(e error) bool {
	_, okp := e.(*errDuplicateKey)
	_, oks := e.(errDuplicateKey)
	return okp || oks
}
func (n errDuplicateKey) Is(e error) bool {
	return IsDuplicateKey(e)
}
func wrapErr(err error, s string, args ...interface{}) errors.Err {
	e := errors.Annotatef(err, s, args...)
	asErr := errors.Err{}
	xerrors.As(e, &asErr)
	return asErr
}

var errFn = func(ss string) func(s string, p ...interface{}) errors.Err {
	fn := func(s string, p ...interface{}) errors.Err {
		return wrapErr(nil, fmt.Sprintf("%s: %s", ss, s), p...)
	}
	return fn
}

var ErrDuplicateObject = func(s string, p ...interface{}) errDuplicateKey {
	return errDuplicateKey{wrapErr(nil, fmt.Sprintf("Duplicate key: %s", s), p...)}
}

type Loader interface {
	s.ActivityLoader
	s.ActorLoader
	s.ObjectLoader
	s.ActivitySaver
	s.ActorSaver
	s.ObjectSaver
}

type loader struct {
	baseURL string
	conn    *pgx.Conn
	d       client.Client
	logFn   loggerFn
	errFn   loggerFn
}

type loggerFn func(logrus.Fields, string, ...interface{})

func logFn(l logrus.FieldLogger, lvl logrus.Level) loggerFn {
	return func(w logrus.Fields, f string, par ...interface{}) {
		if l != nil {
			l.WithFields(w).Logf(lvl, f, par...)
		}
	}
}

func New(conn *pgx.Conn, url string, l logrus.FieldLogger) *loader {
	return &loader{
		conn:    conn,
		baseURL: url,
		d:       client.NewClient(),
		logFn:   logFn(l, logrus.InfoLevel),
		errFn:   logFn(l, logrus.ErrorLevel),
	}
}

func (l loader) Load(f s.Filterable) (as.ItemCollection, int, error) {
	var ff *Filters
	var ok bool
	var total int
	if ff, ok = f.(*Filters); !ok {
		ff = &Filters{}
	}
	ret := make(as.ItemCollection, 0)
	act, total, err := loadFromDb(l.conn, "activities", ff)
	if err != nil {
		ret = append(ret, act...)
	}
	it, total, err := loadFromDb(l.conn, "objects", ff)
	if err != nil {
		ret = append(ret, it...)
	}
	actors, total, err := loadFromDb(l.conn, "actors", ff)
	if err != nil {
		ret = append(ret, actors...)
	}

	return ret, total, err
}
func (l loader) LoadActivities(f s.Filterable) (as.ItemCollection, int, error) {
	var ff *Filters
	var ok bool
	if ff, ok = f.(*Filters); !ok {
		ff = &Filters{}
	}
	return loadFromDb(l.conn, "activities", ff)
}
func (l loader) LoadActors(f s.Filterable) (as.ItemCollection, int, error) {
	var ff *Filters
	var ok bool
	if ff, ok = f.(*Filters); !ok {
		ff = &Filters{}
	}
	return loadFromDb(l.conn, "actors", ff)
}
func (l loader) LoadObjects(f s.Filterable) (as.ItemCollection, int, error) {
	var ff *Filters
	var ok bool
	if ff, ok = f.(*Filters); !ok {
		ff = &Filters{}
	}
	return loadFromDb(l.conn, "objects", ff)
}

func loadFromDb(conn *pgx.Conn, table string, f *Filters) (as.ItemCollection, int, error) {
	clauses, values := f.GetWhereClauses()
	total := 0

	sel := fmt.Sprintf("SELECT id, key, iri, created_at, type, raw FROM %s WHERE %s %s", table, strings.Join(clauses, " AND "), f.GetLimit())
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
		var key Hash
		var iri string
		var createds string
		var typ string
		var raw []byte
		err = rows.Scan(&id, &key, &iri, &createds, &typ, &raw)
		if err != nil {
			return ret, total, errors.Annotatef(err, "scan values error")
		}

		it, err := as.UnmarshalJSON(raw)
		if err != nil {
			return ret, total, errors.Annotatef(err, "unable to unmarshal raw item")
		}
		it = as.FoldProperties(it)
		ret = append(ret, it)
	}

	selCnt := fmt.Sprintf("SELECT COUNT(id) FROM %s WHERE %s", table, strings.Join(clauses, " AND "))
	if err = conn.QueryRow(selCnt, values...).Scan(&total); err != nil {
		err = errors.Annotatef(err, "unable to count all rows")
	}

	return ret, total, err
}

func (l loader) SaveActivity(it as.Item) (as.Item, error) {
	return l.SaveObject(it)
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
	if it.IsLink() {
		// dereference this shit
		iri := it.GetLink()
		if it, err := l.d.LoadIRI(iri); err != nil {
			l.errFn(logrus.Fields{"IRI": iri}, "unable to dereference IRI")
			return it, err
		}
	}

	if as.ValidActivityType(it.GetType()) {
		table = "activities"
		act, err := as.ToActivity(it)
		if err != nil {
			l.errFn(logrus.Fields{"IRI": act.GetLink()}, "unable to load activity")
			return act, err
		}
		// TODO(marius): this whole logic chain needs to be kept separate from the
		//    actual persistence layer, so we don't have to copy/paste it with every new implementation.
		if act.Object != nil {
			if act.Object, err = l.SaveObject(act.Object); err != nil && !IsDuplicateKey(err) {
				l.errFn(logrus.Fields{"IRI": act.GetLink()}, "unable to save activity's object")
				return act, err
			}
			act.Object = as.FoldToIRI(act.Object)
		}
		if act.Actor != nil {
			if act.Actor, err = l.SaveObject(act.Actor); err != nil && !IsDuplicateKey(err) {
				l.errFn(logrus.Fields{"IRI": act.GetLink()}, "unable to save activity's actor")
				return act, err
			}
			act.Actor = as.FoldToIRI(act.Actor)
		}
		if act.Target != nil {
			if act.Target, err = l.SaveObject(act.Target); err != nil && !IsDuplicateKey(err) {
				l.errFn(logrus.Fields{"IRI": act.GetLink()}, "unable to save activity's target")
				return act, err
			}
			act.Target = as.FoldToIRI(act.Target)
		}
		it = act
	} else if as.ValidActorType(it.GetType()) {
		table = "actors"
	} else {
		table = "objects"
	}

	if len(it.GetLink()) > 0 {
		if _, cnt, _ := loadFromDb(l.conn, table, &Filters{
			ItemKey: []Hash{Hash(it.GetLink().String())},
			Type:    []as.ActivityVocabularyType{it.GetType()},
		}); cnt != 0 {
			err := ErrDuplicateObject("%s in table %s", it.GetLink(), table)
			l.errFn(logrus.Fields{
				"table": table,
				"type":  it.GetType(),
				"iri":   it.GetLink(),
				"err":   err.Error(),
			}, "skipping save")
			return it, err
		}
	}
	it, err = l.saveToDb(table, it)
	if err != nil {
		l.errFn(logrus.Fields{
			"table": table,
		}, "%s", err.Error())
	}

	return it, err
}

func (l loader) saveToDb(table string, it as.Item) (as.Item, error) {
	var query string
	//if it.GetID() == nil {
	query = fmt.Sprintf("INSERT INTO %s (key, iri, type, raw) VALUES ($1, $2, $3, $4);", table)
	//} else {
	//	query = fmt.Sprintf("UPDATE %s SET key = $1, iri = $2, type = $3, raw = $4;", table)
	//}

	uuid := uuid2.New()
	iri := it.GetLink().String()
	if len(iri) == 0 {
		// TODO(marius): this needs to be in a different place
		iri = fmt.Sprintf("%s/%s/%s", l.baseURL, table, uuid)
		if as.ValidActivityType(it.GetType()) {
			if a, err := as.ToActivity(it); err == nil {
				a.ID = as.ObjectID(iri)
				it = a
			}
		}
		if as.ValidActorType(it.GetType()) {
			if p, err := activitypub.ToPerson(it); err == nil {
				p.ID = as.ObjectID(iri)
				it = p
			}
		}
		if as.ValidObjectType(it.GetType()) {
			if o, err := as.ToObject(it); err == nil {
				o.ID = as.ObjectID(iri)
				it = o
			}
		}
	}
	raw, _ := jsonld.Marshal(it)

	values := make([]interface{}, 4)
	values[0] = interface{}(uuid)
	values[1] = interface{}(iri)
	values[2] = interface{}(it.GetType())
	values[3] = interface{}(raw)

	_, err := l.conn.Exec(query, values...)
	if err != nil {
		l.errFn(logrus.Fields{
			"err": err.Error(),
		}, "query error")
		return it, errors.Annotatef(err, "query error")
	}

	return it, nil
}
