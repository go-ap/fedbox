package storage

import (
	"fmt"
	s "github.com/go-ap/activitypub/storage"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/fedbox/internal/errors"
	"github.com/jackc/pgx"
	"strings"
	"time"
)

type Paginator = s.Paginator

type Loader struct {
	conn *pgx.Conn
}

func New(conn *pgx.Conn) *Loader {
	return &Loader{
		conn: conn,
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
		var created time.Time
		var typ string
		var raw []byte
		err = rows.Scan(&id, &key, &iri, &created, &typ, &raw)
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
