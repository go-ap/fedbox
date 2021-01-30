// +build storage_sqlite storage_all !sqlite_fs,!storage_boltdb,!storage_badger,!storage_pgx

package sqlite

import (
	"fmt"
	pub "github.com/go-ap/activitypub"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/storage"
	"strings"
)

func getWhereClauses(f storage.Filterable) ([]string, []interface{}) {
	var clauses = make([]string, 0)
	var values = make([]interface{}, 0)

	canHaveAudience := false
	var counter = 1

	url, _ := f.GetLink().URL()
	url.RawQuery = ""
	id := url.String()
	if len(id) > 0 {
		//if canHaveAudience {
		//	// For Link type we need to search the full raw column
		//	clauses = append(clauses, fmt.Sprintf(`"raw"->>'id' = $%d`, counter))
		//}
		clauses = append(clauses, `"iri" = ?`)
		values = append(values, interface{}(id))
		counter++
	}

	if f, ok := f.(storage.FilterableItems); ok {
		types := f.Types()
		if len(types) > 0 {
			keyWhere := make([]string, 0)
			for _, typ := range types {
				canHaveAudience = pub.ActivityTypes.Contains(typ) || pub.ObjectTypes.Contains(typ) || pub.ActorTypes.Contains(typ)

				keyWhere = append(keyWhere, `"type" = ?`)
				values = append(values, interface{}(typ))
				counter++
			}
			clauses = append(clauses, fmt.Sprintf("(%s)", strings.Join(keyWhere, " OR ")))
		}

		//iris := f.IRIs()
		//if len(iris) > 0 {
		//	keyWhere := make([]string, 0)
		//	for _, key := range iris {
		//		if _, err := url.ParseRequestURI(key.String()); err != nil {
		//			// not a valid iri
		//			keyWhere = append(keyWhere, fmt.Sprintf(`"key" ~* $%d`, counter))
		//		} else {
		//			if len(types) == 1 && types[0] == pub.LinkType {
		//				keyWhere = append(keyWhere, fmt.Sprintf(`"raw"::text ~* $%d`, counter))
		//			} else if canHaveAudience {
		//				// For Link type we need to search the full raw column
		//				keyWhere = append(keyWhere, fmt.Sprintf(`"raw"->>'id' = $%d`, counter))
		//			}
		//			keyWhere = append(keyWhere, fmt.Sprintf(`"iri" = $%d`, counter))
		//		}
		//		values = append(values, interface{}(key))
		//		counter++
		//	}
		//	clauses = append(clauses, fmt.Sprintf("(%s)", strings.Join(keyWhere, " OR ")))
		//}
	}

	//if f.To != nil && len(f.To.GetLink()) > 0 {
	//	clauses = append(clauses, fmt.Sprintf(`"raw"->>'to' ~* $%d`, counter))
	//	values = append(values, interface{}(f.To.GetLink()))
	//}
	// TODO(marius): this looks cumbersome - we need to abstract the audience to something easier to query.
	if canHaveAudience {
		keyWhere := make([]string, 0)
		keyWhere = append(keyWhere, fmt.Sprintf(`"raw"->>'to' ~* $%d`, counter))
		keyWhere = append(keyWhere, fmt.Sprintf(`"raw"->>'cc' ~* $%d`, counter))
		keyWhere = append(keyWhere, fmt.Sprintf(`"raw"->>'bto' ~* $%d`, counter))
		keyWhere = append(keyWhere, fmt.Sprintf(`"raw"->>'bcc' ~* $%d`, counter))
		keyWhere = append(keyWhere, fmt.Sprintf(`"raw"->>'audience' ~* $%d`, counter))
		clauses = append(clauses, fmt.Sprintf("(%s)", strings.Join(keyWhere, " OR ")))
		//if f.To == nil {
		//	values = append(values, interface{}(pub.PublicNS))
		//}
	}

	//authors := f.AttributedTo()
	//for _, auth := range authors {
	//	if len(auth) > 0 {
	//		clauses = append(clauses, fmt.Sprintf(`"raw"->>'attributedTo' ~* $%d`, counter))
	//		values = append(values, interface{}(f.Author.GetLink()))
	//	}
	//}

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
