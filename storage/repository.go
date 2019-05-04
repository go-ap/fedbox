package storage

import (
	"fmt"
	s "github.com/go-ap/activitypub/storage"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/fedbox/internal/errors"
	"github.com/go-ap/jsonld"
	uuid2 "github.com/google/uuid"
	"github.com/jackc/pgx"
	"strings"
)

type Paginator = s.Paginator

type Loader struct {
	baseURL string
	conn *pgx.Conn
}

func New(conn *pgx.Conn, url string) *Loader {
	return &Loader{
		conn: conn,
		baseURL: url,
	}
}

func (l Loader) Load(f s.Filterable) (as.ItemCollection, error) {
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
	it, err  := loadFromDb(l.conn, "objects", ff)
	if err != nil {
		ret = append(ret, it...)
	}
	actors, err := loadFromDb(l.conn, "actors", ff)
	if err != nil {
		ret = append(ret, actors...)
	}

	return ret, err
}
func (l Loader) LoadActivities(f s.Filterable) (as.ItemCollection, error) {
	var ff *Filters
	var ok bool
	if ff, ok = f.(*Filters); !ok {
		ff = &Filters{}
	}
	return loadFromDb(l.conn, "activities", ff)
}
func (l Loader) LoadActors(f s.Filterable) (as.ItemCollection, error) {
	var ff *Filters
	var ok bool
	if ff, ok = f.(*Filters); !ok {
		ff = &Filters{}
	}
	return loadFromDb(l.conn, "actors", ff)
}
func (l Loader) LoadObjects(f s.Filterable) (as.ItemCollection, error) {
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

func (l Loader) SaveActivity(it as.Item) (as.Item, error) {
	return l.saveToDb("activities", it)
}
func (l Loader) SaveActor(it as.Item) (as.Item, error) {
	return l.saveToDb("actors", it)
}
func (l Loader) SaveObject(it as.Item) (as.Item, error) {
	return l.saveToDb( "objects", it)
}

func (l Loader)saveToDb(table string, it as.Item) (as.Item, error) {
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
		return it, errors.Annotatef(err, "query error")
	}

	return it, nil
}
