package app

import (
	"fmt"
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/fedbox/activitypub"
	h "github.com/go-ap/handlers"
	"github.com/mariusor/qstring"
	"net/url"
	"path"
	"sync"
)

type iriMap map[pub.IRI]pub.Item

type cache struct {
	enabled bool
	w       sync.RWMutex
	c       iriMap
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
	r.w.Lock()
	defer r.w.Unlock()
	if r.c == nil {
		r.c = make(map[pub.IRI]pub.Item)
	}
	r.c[iri] = it
}

func (r *cache) remove(iris ...pub.IRI) bool {
	if !r.enabled {
		return true
	}
	toInvalidate := pub.IRIs(iris)
	for _, iri := range toInvalidate {
		if h.ValidCollectionIRI(iri) {
			continue
		}
		c := pub.IRI(path.Dir(iri.String()))
		if !toInvalidate.Contains(c) {
			toInvalidate = append(iris, c)
		}
	}
	r.w.Lock()
	defer r.w.Unlock()
	for _, iri := range toInvalidate {
		for key := range r.c {
			// TODO(marius): I need to play around with this a bit
			if key.Contains(iri, false) {
				delete(r.c, key)
			}
		}
	}
	return true
}

func ObjectPurgeCache(caches *cache, o *pub.Object) error {
	if o == nil {
		return nil
	}
	obIRI := o.GetLink()
	if len(obIRI) > 0 {
		//caches.remove(pub.IRI(path.Dir(obIRI.String())))
		caches.remove(obIRI)
	}

	if o.InReplyTo == nil {
		return nil
	}
	if o.InReplyTo.IsCollection() {
		pub.OnCollectionIntf(o.InReplyTo, func(c pub.CollectionInterface) error {
			for _, it := range c.Collection() {
				caches.remove(it.GetLink())
			}
			return nil
		})
	} else {
		caches.remove(o.InReplyTo.GetLink())
	}
	return nil
}

func ActivityPurgeCache(caches *cache, a *pub.Activity, typ h.CollectionType) error {
	for _, r := range a.Recipients() {
		if r.GetLink().Equals(pub.PublicNS, false) {
			continue
		}
		if h.ValidCollectionIRI(r.GetLink()) {
			// TODO(marius): for followers, following collections this should dereference the members
			caches.remove(r.GetLink())
		} else {
			caches.remove(h.Inbox.IRI(r))
		}
	}
	if typ == h.Outbox {
		caches.remove(h.Outbox.IRI(a.Actor))
	}
	if typ == h.Inbox {
		caches.remove(h.Inbox.IRI(a.Actor))
	}

	pub.OnObject(a.Object, func(o *pub.Object) error {
		return ObjectPurgeCache(caches, o)
	})

	aIRI := a.GetLink()
	if len(aIRI) > 0 {
		caches.remove(aIRI)
	}
	return nil
}
