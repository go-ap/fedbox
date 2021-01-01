package cache

import (
	pub "github.com/go-ap/activitypub"
	h "github.com/go-ap/handlers"
	"path"
	"sync"
)

type (
	iriMap map[pub.IRI]pub.Item
	keyFn func(interface{}) pub.IRI
	store struct {
		enabled bool
		w       sync.RWMutex
		c       iriMap
		hashFn  keyFn
	}
	CanStore interface {
		Set(iri pub.IRI, it pub.Item)
		Get(iri pub.IRI) pub.Item
		Remove(iris ...pub.IRI) bool
	}
)

func New(enabled bool, ) *store {
	return &store{enabled: enabled, c: make(iriMap)}
}

func (r *store) Get(iri pub.IRI) pub.Item {
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

func (r *store) Set(iri pub.IRI, it pub.Item) {
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

func (r *store) Remove(iris ...pub.IRI) bool {
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

func ActivityPurge(cache CanStore, a *pub.Activity, typ h.CollectionType) error {
	toRemove := make(pub.IRIs, 0)
	err := aggregateActivityIRIs(&toRemove, a, typ)
	if err != nil {
		return err
	}
	if len(toRemove) > 0 {
		cache.Remove(toRemove...)
	}
	return nil
}
