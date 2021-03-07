// +build storage_sqlite storage_all !sqlite_fs,!storage_boltdb,!storage_badger,!storage_pgx

package sqlite

import (
	"fmt"
	pub "github.com/go-ap/activitypub"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/handlers"
	"github.com/go-ap/storage"
	"path"
	"strings"
)

func isCollection(col string) bool {
	return col == string(ap.ActorsType) || col == string(ap.ActivitiesType) || col == string(ap.ObjectsType)
}

func getStringFieldInJSONWheres(strs ap.CompStrs, props ...string) (string, []interface{}) {
	if len(strs) == 0 {
		return "", nil
	}
	var values = make([]interface{}, 0)
	keyWhere := make([]string, 0)
	for _, n := range strs {
		switch n.Operator {
		case "!":
			for _, prop := range props {
				if n.Str == string(pub.NilLangRef) {
					keyWhere = append(keyWhere, fmt.Sprintf(`json_extract("raw", '$.%s') IS NOT NULL`, prop))
				} else {
					keyWhere = append(keyWhere, fmt.Sprintf(`json_extract("raw", '$.%s') != ?`, prop))
					values = append(values, interface{}(n.Str))
				}
			}
		case "~":
			for _, prop := range props {
				keyWhere = append(keyWhere, fmt.Sprintf(`json_extract("raw", '$.%s') LIKE ?`, prop))
				values = append(values, interface{}("%"+n.Str+"%"))
			}
		case "", "=":
			fallthrough
		default:
			for _, prop := range props {
				if n.Str == string(pub.NilLangRef) {
					keyWhere = append(keyWhere, fmt.Sprintf(`json_extract("raw", '$.%s') IS NULL`, prop))
				} else {
					keyWhere = append(keyWhere, fmt.Sprintf(`json_extract("raw", '$.%s') = ?`, prop))
					values = append(values, interface{}(n.Str))
				}
			}
		}
	}
	return fmt.Sprintf("(%s)", strings.Join(keyWhere, " OR ")), values
}

func getStringFieldWheres(strs ap.CompStrs, fields ...string) (string, []interface{}) {
	if len(strs) == 0 {
		return "", nil
	}
	var values = make([]interface{}, 0)
	keyWhere := make([]string, 0)
	for _, t := range strs {
		switch t.Operator {
		case "!":
			for _, field := range fields {
				if t.Str == string(pub.NilLangRef) {
					keyWhere = append(keyWhere, fmt.Sprintf(`"%s" IS NOT NULL`, field))
				} else {
					keyWhere = append(keyWhere, fmt.Sprintf(`"%s" != ?`, field))
					values = append(values, interface{}(t.Str))
				}
			}
		case "~":
			for _, field := range fields {
				keyWhere = append(keyWhere, fmt.Sprintf(`"%s" LIKE ?`, field))
				values = append(values, interface{}("%"+t.Str+"%"))
			}
		case "", "=":
			for _, field := range fields {
				if t.Str == string(pub.NilLangRef) {
					keyWhere = append(keyWhere, fmt.Sprintf(`"%s" IS NULL`, field))
				} else {
					keyWhere = append(keyWhere, fmt.Sprintf(`"%s" = ?`, field))
					values = append(values, interface{}(t.Str))
				}
			}
		}
	}

	return fmt.Sprintf("(%s)", strings.Join(keyWhere, " OR ")), values
}

func getTypeWheres(strs ap.CompStrs) (string, []interface{}) {
	return getStringFieldWheres(strs, "type")
}

func getURLWheres(strs ap.CompStrs) (string, []interface{}) {
	clause, values := getStringFieldWheres(strs, "url")
	jClause, jValues := getStringFieldInJSONWheres(strs, "url")
	if len(jClause) > 0 {
		if len(clause) > 0 {
			clause += " OR "
		}
		clause += jClause
	}
	values = append(values, jValues...)
	return clause, values
}

var MandatoryCollections = handlers.CollectionTypes{
	handlers.Inbox,
	handlers.Outbox,
}

func getIRIWheres(strs ap.CompStrs, id pub.IRI) (string, []interface{}) {
	iriClause, iriValues := getStringFieldWheres(strs, "iri")

	skipId := strings.Contains(iriClause, `"iri" = ?`)
	if skipId {
		return iriClause, iriValues
	}

	if u, _ := id.URL(); u != nil {
		u.RawQuery = ""
		id = pub.IRI(u.String())
	}
	// FIXME(marius): this is a hack that avoids trying to use clause on IRI, when iri == "/"
	if len(id) > 1 {
		if len(iriClause) > 0 {
			iriClause += " OR "
		}
		if base := path.Base(id.String()); isCollection(base) {
			iriClause += `"iri" LIKE ?`
			iriValues = append(iriValues, interface{}("%"+id+"%"))
		} else {
			iriClause += `"iri" = ?`
			iriValues = append(iriValues, interface{}(id))
		}
	}
	return iriClause, iriValues
}

func getNamesWheres(strs ap.CompStrs) (string, []interface{}) {
	return getStringFieldInJSONWheres(strs, "name", "preferredUsername")
}

func getInReplyToWheres(strs ap.CompStrs) (string, []interface{}) {
	return getStringFieldInJSONWheres(strs, "inReplyTo")
}

func getAttributedToWheres(strs ap.CompStrs) (string, []interface{}) {
	return getStringFieldInJSONWheres(strs, "attributedTo")
}

func getWhereClauses(f *ap.Filters) ([]string, []interface{}) {
	var clauses = make([]string, 0)
	var values = make([]interface{}, 0)

	if typClause, typValues := getTypeWheres(f.Types()); len(typClause) > 0 {
		values = append(values, typValues...)
		clauses = append(clauses, typClause)
	}

	if iriClause, iriValues := getIRIWheres(f.IRIs(), f.GetLink()); len(iriClause) > 0 {
		values = append(values, iriValues...)
		clauses = append(clauses, iriClause)
	}

	if nameClause, nameValues := getNamesWheres(f.Names()); len(nameClause) > 0 {
		values = append(values, nameValues...)
		clauses = append(clauses, nameClause)
	}

	if replClause, replValues := getInReplyToWheres(f.InReplyTo()); len(replClause) > 0 {
		values = append(values, replValues...)
		clauses = append(clauses, replClause)
	}

	if authorClause, authorValues := getAttributedToWheres(f.AttributedTo()); len(authorClause) > 0 {
		values = append(values, authorValues...)
		clauses = append(clauses, authorClause)
	}

	if urlClause, urlValues := getURLWheres(f.URLs()); len(urlClause) > 0 {
		values = append(values, urlValues...)
		clauses = append(clauses, urlClause)
	}

	if len(clauses) == 0 {
		if ap.FedboxCollections.Contains(f.Collection) {
			clauses = append(clauses, " true")
		} else {
			clauses = append(clauses, " false")
		}
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
