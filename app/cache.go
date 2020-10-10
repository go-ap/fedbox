package app

import (
	"fmt"
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/fedbox/activitypub"
	"github.com/mariusor/qstring"
	"net/url"
	"path"
)

type reqCache map[pub.IRI]pub.Item

func cacheKey(f *activitypub.Filters) pub.IRI {
	var iri pub.IRI

	if q, err := qstring.Marshal(f); err == nil && len(q) > 0 {
		iri = pub.IRI(fmt.Sprintf("%s?%s", f.GetLink(), q.Encode()))
	} else {
		iri = f.GetLink()
	}
	u, _ := iri.URL()
	if auth := f.Authenticated; auth != nil && !auth.ID.Equals(pub.PublicNS, true) {
		u.User = url.User(path.Base(f.Authenticated.ID.String()))
	}
	return pub.IRI(u.String())
}

func (r reqCache) has(iri pub.IRI) bool {
	_, ok := r[iri]
	return ok
}

func (r reqCache) get(iri pub.IRI) pub.Item {
	if it, ok := r[iri]; ok {
		return it
	}
	return nil
}

func (r *reqCache) set(iri pub.IRI, it pub.Item) {
	(*r)[iri] = it
}

func (r *reqCache) remove(iri pub.IRI) bool {
	if iri == pub.PublicNS {
		return false
	}
	for key := range *r {
		// TODO(marius): I need to play around with this a bit
		if key.Contains(iri, false) {
			delete(*r, key)
		}
	}
	return true
}
