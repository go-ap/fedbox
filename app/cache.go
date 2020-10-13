package app

import (
	"fmt"
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/fedbox/activitypub"
	"github.com/mariusor/qstring"
	"net/url"
	"path"
	"sync"
)

type iriMap map[pub.IRI]pub.Item

type cache struct {
	enabled bool
	w sync.RWMutex
	c iriMap
}

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

func (r *cache) get(iri pub.IRI) pub.Item {
	if !r.enabled {
		return nil
	}
	r.w.RLock()
	defer r.w.RUnlock()
	if it, ok := r.c[iri]; ok {
		return it
	}
	return nil
}

func (r *cache) set(iri pub.IRI, it pub.Item) {
	if !r.enabled {
		return
	}
	if r.c == nil {
		r.c = make(map[pub.IRI]pub.Item)
	}
	r.w.Lock()
	defer r.w.Unlock()
	r.c[iri] = it
}

func (r *cache) remove(iri pub.IRI) bool {
	if !r.enabled {
		return true
	}
	if iri == pub.PublicNS {
		return false
	}
	r.w.Lock()
	defer r.w.Unlock()
	for key := range r.c {
		// TODO(marius): I need to play around with this a bit
		if key.Contains(iri, false) {
			delete(r.c, key)
		}
	}
	return true
}
