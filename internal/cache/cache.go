package cache

import (
	"path"
	"sync"

	pub "github.com/go-ap/activitypub"
	h "github.com/go-ap/handlers"
)

type (
	iriMap map[pub.IRI]pub.Item
	store  struct {
		enabled bool
		w       sync.RWMutex
		c       iriMap
	}
	CanStore interface {
		Set(iri pub.IRI, it pub.Item)
		Get(iri pub.IRI) pub.Item
		Remove(iris ...pub.IRI) bool
	}
)

func New(enabled bool) *store {
	return &store{enabled: enabled, c: make(iriMap)}
}

func (r *store) Get(iri pub.IRI) pub.Item {
	if r == nil || !r.enabled {
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
	if r == nil || !r.enabled {
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
	if r == nil || !r.enabled {
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

func removeAccum(toRemove *pub.IRIs, iri pub.IRI, col h.CollectionType) {
	if repl := col.IRI(iri); !toRemove.Contains(repl) {
		*toRemove = append(*toRemove, repl)
	}
}

func accumForProperty(it pub.Item, toRemove *pub.IRIs, col h.CollectionType) {
	if pub.IsNil(it) {
		return
	}
	if pub.IsItemCollection(it) {
		pub.OnItemCollection(it, func(c *pub.ItemCollection) error {
			for _, ob := range c.Collection() {
				removeAccum(toRemove, ob.GetLink(), col)
			}
			return nil
		})
	} else {
		removeAccum(toRemove, it.GetLink(), col)
	}
}

func aggregateItemIRIs(toRemove *pub.IRIs, it pub.Item) error {
	if it == nil {
		return nil
	}
	if obIRI := it.GetLink(); len(obIRI) > 0 && !toRemove.Contains(obIRI) {
		*toRemove = append(*toRemove, obIRI)
	}
	if !it.IsObject() {
		return nil
	}
	return pub.OnObject(it, func(o *pub.Object) error {
		accumForProperty(o.InReplyTo, toRemove, h.Replies)
		accumForProperty(o.AttributedTo, toRemove, h.Outbox)
		return nil
	})
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
			accumForProperty(r, toRemove, h.Inbox)
		}
	}
	if destCol := typ.IRI(a.Actor); !toRemove.Contains(destCol) {
		*toRemove = append(*toRemove, destCol)
	}

	if aIRI := a.GetLink(); len(aIRI) > 0 && !toRemove.Contains(aIRI) {
		*toRemove = append(*toRemove, aIRI)
	}

	withSideEffects := pub.ActivityVocabularyTypes{pub.UpdateType, pub.UndoType, pub.DeleteType}
	if withSideEffects.Contains(a.GetType()) {
		base := path.Dir(a.Object.GetLink().String())
		*toRemove = append(*toRemove, pub.IRI(base))
		*toRemove = append(*toRemove, a.Object.GetLink())
	}

	return aggregateItemIRIs(toRemove, a.Object)
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
