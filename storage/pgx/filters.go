//go:build storage_pgx || storage_all || (!storage_boltdb && !storage_fs && !storage_badger && !storage_sqlite)

package pgx

import (
	"fmt"
	"strings"

	vocab "github.com/go-ap/activitypub"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/processing"
)

func getWhereClauses(f processing.Filterable) ([]string, []interface{}) {
	var clauses = make([]string, 0)
	var values = make([]interface{}, 0)

	canHaveAudience := false
	var counter = 1

	id := f.GetLink()
	if len(id) > 0 {
		//if canHaveAudience {
		//	// For Link type we need to search the full raw column
		//	clauses = append(clauses, fmt.Sprintf(`"raw"->>'id' = $%d`, counter))
		//}
		clauses = append(clauses, fmt.Sprintf(`"iri" = $%d`, counter))
		values = append(values, interface{}(id))
		counter++
	}
	//
	//if f, ok := f.(ap.Filters); ok {
	//	keys := f.IRIs()
	//	if len(keys) > 0 {
	//		keyWhere := make([]string, 0)
	//		for _, hash := range keys {
	//			keyWhere = append(keyWhere, fmt.Sprintf(`"key" ~* $%d`, counter))
	//			values = append(values, interface{}(hash))
	//			counter++
	//		}
	//		clauses = append(clauses, fmt.Sprintf("(%s)", strings.Join(keyWhere, " OR ")))
	//	}
	//}

	if f, ok := f.(processing.FilterableItems); ok {
		types := f.Types()
		if len(types) > 0 {
			keyWhere := make([]string, 0)
			for _, typ := range types {
				canHaveAudience = vocab.ActivityTypes.Contains(typ) || vocab.ObjectTypes.Contains(typ) || vocab.ActorTypes.Contains(typ)

				keyWhere = append(keyWhere, fmt.Sprintf(`"type" = $%d`, counter))
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
		//			if len(types) == 1 && types[0] == vocab.LinkType {
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
		//	values = append(values, interface{}(vocab.PublicNS))
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

func getLimit(f processing.Filterable) string {
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
