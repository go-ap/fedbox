package storage

import (
	"fmt"
	s "github.com/go-ap/activitypub/storage"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/fedbox/internal/errors"
	"github.com/go-ap/jsonld"
	uuid2 "github.com/google/uuid"
	"github.com/jackc/pgx"
	"github.com/sirupsen/logrus"
	"strings"
)

type Paginator = s.Paginator

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
		logFn: logFn(l, logrus.InfoLevel),
		errFn: logFn(l, logrus.ErrorLevel),
	}
}

func (l loader) Load(f s.Filterable) (as.ItemCollection, error) {
	var ff *Filters
	var ok bool
	if ff, ok = f.(*Filters); !ok {
		ff = &Filters{}
	}
	ret := make(as.ItemCollection, 0)
	act, err := loadFromDb(l.conn, "activities", ff)
	if err != nil {
		ret = append(ret, act...)
	}
	it, err := loadFromDb(l.conn, "objects", ff)
	if err != nil {
		ret = append(ret, it...)
	}
	actors, err := loadFromDb(l.conn, "actors", ff)
	if err != nil {
		ret = append(ret, actors...)
	}

	return ret, err
}
func (l loader) LoadActivities(f s.Filterable) (as.ItemCollection, error) {
	var ff *Filters
	var ok bool
	if ff, ok = f.(*Filters); !ok {
		ff = &Filters{}
	}
	return loadFromDb(l.conn, "activities", ff)
}
func (l loader) LoadActors(f s.Filterable) (as.ItemCollection, error) {
	var ff *Filters
	var ok bool
	if ff, ok = f.(*Filters); !ok {
		ff = &Filters{}
	}
	return loadFromDb(l.conn, "actors", ff)
}
func (l loader) LoadObjects(f s.Filterable) (as.ItemCollection, error) {
	var ff *Filters
	var ok bool
	if ff, ok = f.(*Filters); !ok {
		ff = &Filters{}
	}
	return loadFromDb(l.conn, "objects", ff)
}

func loadFromDb(conn *pgx.Conn, table string, f *Filters) (as.ItemCollection, error) {
	clauses, values := f.GetWhereClauses()

	sel := fmt.Sprintf("SELECT id, key, iri, created_at, type, raw FROM %s WHERE %s %s", table, strings.Join(clauses, " AND "), f.GetLimit())
	rows, err := conn.Query(sel, values...)
	defer rows.Close()
	if err != nil {
		if err == pgx.ErrNoRows {
			return as.ItemCollection{}, nil
		}
		return nil, errors.Annotatef(err, "query error")
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
			return ret, err
		}
		it, err := as.UnmarshalJSON(raw)
		if err != nil {
			return ret, err
		}
		ret = append(ret, it)
	}

	return ret, nil
}

func (l loader) SaveActivity(it as.Item) (as.Item, error) {
	return l.SaveObject(it)
}
func (l loader) SaveActor(it as.Item) (as.Item, error) {
	return l.SaveObject(it)
}
func (l loader) SaveObject(it as.Item) (as.Item, error) {
	var err error
	var table string
	if as.ValidActivityType(it.GetType()) {
		table = "activities"
		act := as.ToActivity(it)
		act.Object, err = l.SaveObject(act.Object)
		act.Actor, err = l.SaveObject(act.Actor)
		it = act
	} else if as.ValidActorType(it.GetType()) {
		table = "actors"
	} else if as.ValidObjectType(it.GetType()) {
		table = "objects"
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
		iri = fmt.Sprintf("%s/%s/%s", l.baseURL, table, uuid)
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
