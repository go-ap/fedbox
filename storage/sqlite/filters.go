// +build storage_sqlite storage_all !sqlite_fs,!storage_boltdb,!storage_badger,!storage_pgx

package sqlite

import (
	"fmt"
	pub "github.com/go-ap/activitypub"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/storage"
	"strings"
)


func getWhereClauses(f *ap.Filters) ([]string, []interface{}) {
	var clauses = make([]string, 0)
	var values = make([]interface{}, 0)

	var counter = 1

	types := f.Types()
	if len(types) > 0 {
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
				keyWhere = append(keyWhere, `"iri" = ?`)
			}
			values = append(values, interface{}(key))
			counter++
		}
		clauses = append(clauses, fmt.Sprintf("(%s)", strings.Join(keyWhere, " OR ")))
	} else {
		u, _ := f.GetLink().URL()
		u.RawQuery = ""
		id := u.String()
		if len(id) > 0 {
			clauses = append(clauses, `"iri" = ?`)
			values = append(values, interface{}(id))
			counter++
		}
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
