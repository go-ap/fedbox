package pgx

import (
	"fmt"
	as "github.com/go-ap/activitystreams"
	ap "github.com/go-ap/fedbox/activitypub"
	"net/url"
	"strings"
)

func getWhereClauses(f ap.Filters) ([]string, []interface{}) {
	var clauses = make([]string, 0)
	var values = make([]interface{}, 0)

	var counter = 1

	keys := f.Key
	if len(keys) > 0 {
		keyWhere := make([]string, 0)
		for _, hash := range keys {
			keyWhere = append(keyWhere, fmt.Sprintf(`"key" ~* $%d`, counter))
			values = append(values, interface{}(hash))
			counter++
		}
		clauses = append(clauses, fmt.Sprintf("(%s)", strings.Join(keyWhere, " OR ")))
	}
	types := f.Types()
	if len(types) > 0 {
		keyWhere := make([]string, 0)
		for _, typ := range types {
			keyWhere = append(keyWhere, fmt.Sprintf(`"type" = $%d`, counter))
			values = append(values, interface{}(typ))
			counter++
		}
		clauses = append(clauses, fmt.Sprintf("(%s)", strings.Join(keyWhere, " OR ")))
	}

	canHaveAudience := false
	for _, typ := range f.Type {
		canHaveAudience = as.ActivityTypes.Contains(typ) || as.ObjectTypes.Contains(typ) || as.ActorTypes.Contains(typ)
	}

	iris := f.IRIs()
	if len(iris) > 0 {
		keyWhere := make([]string, 0)
		for _, key := range iris {
			if _, err := url.ParseRequestURI(key.String()); err != nil {
				// not a valid iri
				keyWhere = append(keyWhere, fmt.Sprintf(`"key" ~* $%d`, counter))
			} else {
				if len(f.Type) == 1 && f.Type[0] == as.LinkType {
					keyWhere = append(keyWhere, fmt.Sprintf(`"raw"::text ~* $%d`, counter))
				} else if canHaveAudience {
					// For Link type we need to search the full raw column
					keyWhere = append(keyWhere, fmt.Sprintf(`"raw"->>'id' = $%d`, counter))
				}
				keyWhere = append(keyWhere, fmt.Sprintf(`"iri" = $%d`, counter))
			}
			values = append(values, interface{}(key))
			counter++
		}
		clauses = append(clauses, fmt.Sprintf("(%s)", strings.Join(keyWhere, " OR ")))
	}

	if len(f.IRI) > 0 {
		if canHaveAudience {
			// For Link type we need to search the full raw column
			clauses = append(clauses, fmt.Sprintf(`"raw"->>'id' = $%d`, counter))
		}
		clauses = append(clauses, fmt.Sprintf(`"iri" = $%d`, counter))
		values = append(values, interface{}(f.IRI))
		counter++
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
		if f.To == nil {
			values = append(values, interface{}(ap.ActivityStreamsPublicNS))
		}
	}

	if f.Author != nil && len(f.Author.GetLink()) > 0 {
		clauses = append(clauses, fmt.Sprintf(`"raw"->>'attributedTo' ~* $%d`, counter))
		values = append(values, interface{}(f.Author.GetLink()))
	}

	if len(clauses) == 0 {
		clauses = append(clauses, " true")
	}

	return clauses, values
}

func getLimit(f ap.Filters) string {
	if f.MaxItems == 0 {
		return ""
	}
	limit := fmt.Sprintf(" LIMIT %d", f.MaxItems)
	if f.CurPage > 0 {
		limit = fmt.Sprintf("%s OFFSET %d", limit, f.MaxItems*(f.CurPage-1))
	}
	return limit
}
