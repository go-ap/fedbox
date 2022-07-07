package cache

import (
	"path"
	"sync"

	vocab "github.com/go-ap/activitypub"
)

type (
	iriMap map[vocab.IRI]vocab.Item
	store  struct {
		enabled bool
		w       sync.RWMutex
		c       iriMap
	}
	CanStore interface {
		Set(iri vocab.IRI, it vocab.Item)
		Get(iri vocab.IRI) vocab.Item
		Remove(iris ...vocab.IRI) bool
	}
)

func New(enabled bool) *store {
	return &store{enabled: enabled, c: make(iriMap)}
}

func (r *store) Get(iri vocab.IRI) vocab.Item {
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

func (r *store) Set(iri vocab.IRI, it vocab.Item) {
	if r == nil || !r.enabled {
		return
	}
	r.w.Lock()
	defer r.w.Unlock()
	if r.c == nil {
		r.c = make(map[vocab.IRI]vocab.Item)
	}
	r.c[iri] = it
}

func (r *store) Clear() {
	if r == nil || !r.enabled {
		return
	}
}

func (r *store) Remove(iris ...vocab.IRI) bool {
	if r == nil || !r.enabled {
		return true
	}
	if len(iris) == 0 {
		for key := range r.c {
			delete(r.c, key)
		}
		return true
	}
	toInvalidate := vocab.IRIs(iris)
	for _, iri := range iris {
		if vocab.ValidCollectionIRI(iri) {
			continue
		}
		c := vocab.IRI(path.Dir(iri.String()))
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

func removeAccum(toRemove *vocab.IRIs, iri vocab.IRI, col vocab.CollectionPath) {
	if repl := col.IRI(iri); !toRemove.Contains(repl) {
		*toRemove = append(*toRemove, repl)
	}
}

func accumForProperty(it vocab.Item, toRemove *vocab.IRIs, col vocab.CollectionPath) {
	if vocab.IsNil(it) {
		return
	}
	if vocab.IsItemCollection(it) {
		vocab.OnItemCollection(it, func(c *vocab.ItemCollection) error {
			for _, ob := range c.Collection() {
				removeAccum(toRemove, ob.GetLink(), col)
			}
			return nil
		})
	} else {
		removeAccum(toRemove, it.GetLink(), col)
	}
}

func aggregateItemIRIs(toRemove *vocab.IRIs, it vocab.Item) error {
	if it == nil {
		return nil
	}
	if obIRI := it.GetLink(); len(obIRI) > 0 && !toRemove.Contains(obIRI) {
		*toRemove = append(*toRemove, obIRI)
	}
	if !it.IsObject() {
		return nil
	}
	return vocab.OnObject(it, func(o *vocab.Object) error {
		accumForProperty(o.InReplyTo, toRemove, vocab.Replies)
		accumForProperty(o.AttributedTo, toRemove, vocab.Outbox)
		return nil
	})
}

func aggregateActivityIRIs(toRemove *vocab.IRIs, a *vocab.Activity, typ vocab.CollectionPath) error {
	for _, r := range a.Recipients() {
		if r.GetLink().Equals(vocab.PublicNS, false) {
			continue
		}
		if iri := r.GetLink(); vocab.ValidCollectionIRI(iri) {
			// TODO(marius): for followers, following collections this should dereference the members
			if !toRemove.Contains(iri) {
				*toRemove = append(*toRemove, iri)
			}
		} else {
			accumForProperty(r, toRemove, vocab.Inbox)
		}
	}
	if destCol := typ.IRI(a.Actor); !toRemove.Contains(destCol) {
		*toRemove = append(*toRemove, destCol)
	}

	if aIRI := a.GetLink(); len(aIRI) > 0 && !toRemove.Contains(aIRI) {
		*toRemove = append(*toRemove, aIRI)
	}

	withSideEffects := vocab.ActivityVocabularyTypes{vocab.UpdateType, vocab.UndoType, vocab.DeleteType}
	if withSideEffects.Contains(a.GetType()) {
		base := path.Dir(a.Object.GetLink().String())
		*toRemove = append(*toRemove, vocab.IRI(base))
		*toRemove = append(*toRemove, a.Object.GetLink())
	}

	return aggregateItemIRIs(toRemove, a.Object)
}

func ActivityPurge(cache CanStore, a *vocab.Activity, iri vocab.IRI) error {
	toRemove := make(vocab.IRIs, 0)
	_, typ := vocab.Split(iri)
	err := aggregateActivityIRIs(&toRemove, a, typ)
	if err != nil {
		return err
	}
	if len(toRemove) > 0 {
		cache.Remove(toRemove...)
	}
	return nil
}
