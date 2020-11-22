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
	for _, iri := range iris {
		if h.ValidCollectionIRI(iri) {
			continue
		}
		c := pub.IRI(path.Dir(iri.String()))
		if !toInvalidate.Contains(c) {
			toInvalidate = append(toInvalidate, c)
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

func aggregateObjectIRIs(toRemove *pub.IRIs, o *pub.Object) error {
	if o == nil {
		return nil
	}
	obIRI := o.GetLink()
	if len(obIRI) > 0 && !toRemove.Contains(obIRI){
		*toRemove = append(*toRemove, obIRI)
	}

	if o.InReplyTo == nil {
		return nil
	}
	if o.InReplyTo.IsCollection() {
		pub.OnCollectionIntf(o.InReplyTo, func(c pub.CollectionInterface) error {
			for _, it := range c.Collection() {
				if !toRemove.Contains(it.GetLink()) {
					*toRemove = append(*toRemove, it.GetLink())
				}
			}
			return nil
		})
	} else {
		if !toRemove.Contains(o.InReplyTo.GetLink()) {
			*toRemove = append(*toRemove, o.InReplyTo.GetLink())
		}
	}
	return nil
}

func aggregateActivityIRIs(toRemove *pub.IRIs, a *pub.Activity, typ h.CollectionType) error {
	for _, r := range a.Recipients() {
		if r.GetLink().Equals(pub.PublicNS, false) {
			continue
		}
		if iri := r.GetLink(); h.ValidCollectionIRI(iri) {
			// TODO(marius): for followers, following collections this should dereference the members
			if !toRemove.Contains(iri) {
				*toRemove = append(*toRemove, iri)
			}
		} else {
			if inbox := h.Inbox.IRI(r); !toRemove.Contains(inbox) {
				*toRemove = append(*toRemove, inbox)
			}
		}
	}
	if destCol := typ.IRI(a.Actor); !toRemove.Contains(destCol) {
		*toRemove = append(*toRemove, destCol)
	}

	pub.OnObject(a.Object, func(o *pub.Object) error {
		return aggregateObjectIRIs(toRemove, o)
	})

	if aIRI := a.GetLink(); len(aIRI) > 0 && !toRemove.Contains(aIRI) {
		*toRemove = append(*toRemove, aIRI)
	}
	return nil
}

func ActivityPurgeCache(caches *cache, a *pub.Activity, typ h.CollectionType) error {
	toRemove := make(pub.IRIs, 0)
	err := aggregateActivityIRIs(&toRemove, a, typ)
	if err != nil {
		return err
	}
	if len(toRemove) > 0 {
		caches.remove(toRemove...)
	}
	return nil
}
