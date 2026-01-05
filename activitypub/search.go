package ap

import (
	"net/url"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/filters"
)

func ByType(types ...vocab.ActivityVocabularyType) url.Values {
	q := make(url.Values)
	tt := make([]string, len(types))
	for i, t := range types {
		tt[i] = string(t)
	}
	q["type"] = tt
	return q
}

func ByName(names ...string) url.Values {
	q := make(url.Values)
	q["name"] = names
	return q
}

func ByURL(urls ...vocab.IRI) url.Values {
	q := make(url.Values)
	uu := make([]string, len(urls))
	for i, u := range urls {
		uu[i] = u.String()
	}
	q["url"] = uu
	return q
}

func IRIWithFilters(iri vocab.IRI, searchParams ...url.Values) vocab.IRI {
	q := make(url.Values)
	for _, params := range searchParams {
		for k, vals := range params {
			if _, ok := q[k]; !ok {
				q[k] = make([]string, 0)
			}
			q[k] = append(q[k], vals...)
		}
	}
	if s, err := iri.URL(); err == nil {
		s.RawQuery = q.Encode()
		iri = vocab.IRI(s.String())
	}
	return iri
}

func SearchActorsIRI(baseIRI vocab.IRI, searchParams ...url.Values) vocab.IRI {
	return IRIWithFilters(filters.ActorsType.IRI(baseIRI), searchParams...)
}
