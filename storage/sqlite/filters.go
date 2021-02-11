// +build storage_sqlite storage_all !sqlite_fs,!storage_boltdb,!storage_badger,!storage_pgx

package sqlite

import (
	"fmt"
	pub "github.com/go-ap/activitypub"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/storage"
	"path"
	"strings"
)

func isCollection(col string) bool {
	return col == string(ap.ActorsType) || col == string(ap.ActivitiesType) || col == string(ap.ObjectsType)
}

func getWhereClauses(f *ap.Filters) ([]string, []interface{}) {
	var clauses = make([]string, 0)
	var values = make([]interface{}, 0)

	var counter = 1

	if types := f.Types(); len(types) > 0 {
		keyWhere := make([]string, 0)
		for _, t := range types {
			typ := pub.ActivityVocabularyType(t.Str)
			switch t.Operator {
			case "!":
				keyWhere = append(keyWhere, `"type" != ?`)
			case "~":
				keyWhere = append(keyWhere, `"type" LIKE ?`)
			case "", "=":
				keyWhere = append(keyWhere, `"type" = ?`)
			}

			values = append(values, interface{}(typ))
			counter++
		}
		clauses = append(clauses, fmt.Sprintf("(%s)", strings.Join(keyWhere, " OR ")))
	}

	iris := f.IRIs()
	skipId := false
	if len(iris) > 0 {
		keyWhere := make([]string, 0)
		for _, iriF := range iris {
			key := iriF.Str
			switch iriF.Operator {
			case "!":
				keyWhere = append(keyWhere, `iri != ?`)
			case "~":
				keyWhere = append(keyWhere, `iri LIKE ?`)
			case "", "=":
				skipId = true
				keyWhere = append(keyWhere, `"iri" = ?`)
			}
			values = append(values, interface{}(key))
			counter++
		}
		clauses = append(clauses, fmt.Sprintf("(%s)", strings.Join(keyWhere, " OR ")))
	}
	id := f.GetLink()
	if u, _ := id.URL(); u != nil {
		u.RawQuery = ""
		id = pub.IRI(u.String())
	}
	if len(id) > 0 && !skipId {
		if base := path.Base(id.String()); isCollection(base) {
			clauses = append(clauses, `"iri" like ?`)
			values = append(values, interface{}(id+"%"))
			counter++
		} else {
			clauses = append(clauses, `"iri" = ?`)
			values = append(values, interface{}(id))
			counter++
		}
	}

	if names := f.Names(); len(names) > 0 {
		keyWhere := make([]string, 0)
		for _, n := range names {
			switch n.Operator {
			case "!":
				keyWhere = append(keyWhere, `json_extract("raw", '$.name') != ? or json_extract("raw", '$.preferredUsername') != ? `)
				values = append(values, interface{}(n.Str), interface{}(n.Str))
			case "~":
				keyWhere = append(keyWhere, `json_extract("raw", '$.name') LIKE ? or json_extract("raw", '$.preferredUsername') LIKE ? `)
				values = append(values, interface{}(n.Str+"%"), interface{}(n.Str+"%"))
			case "", "=":
				keyWhere = append(keyWhere, `json_extract("raw", '$.name') = ? or json_extract("raw", '$.preferredUsername') = ?`)
				values = append(values, interface{}(n.Str), interface{}(n.Str))
			}
			counter++
		}
		clauses = append(clauses, fmt.Sprintf("(%s)", strings.Join(keyWhere, " OR ")))
	}
	if len(clauses) == 0 {
		clauses = append(clauses, " true")
	}

	return clauses, values
}

func getLimit(f storage.Filterable) string {
	if f, ok := f.(*ap.Filters); ok {
		if f.MaxItems == 0 {
			return ""
		}
		limit := fmt.Sprintf(" LIMIT %d", f.MaxItems)
		if f.CurPage > 0 {
			return fmt.Sprintf("%s OFFSET %d", limit, f.MaxItems*(f.CurPage-1))
		}
	}
	return ""
}
