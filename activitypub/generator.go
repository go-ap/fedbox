package ap

import (
	"time"

	"git.sr.ht/~mariusor/storage-all"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	"github.com/openshift/osin"
	"github.com/pborman/uuid"
)

// GenerateID generates a unique identifier for the 'it' [vocab.Item].
func GenerateID(it vocab.Item, partOf vocab.IRI, by vocab.Item) (vocab.ID, error) {
	uid := uuid.New()
	id := partOf.GetLink().AddPath(uid)
	typ := it.GetType()
	if vocab.ActivityTypes.Contains(typ) || vocab.IntransitiveActivityTypes.Contains(typ) {
		err := vocab.OnIntransitiveActivity(it, func(a *vocab.IntransitiveActivity) error {
			if rec := a.Recipients(); rec.Contains(vocab.PublicNS) {
				return nil
			}
			if vocab.IsNil(by) {
				by = a.Actor
			}
			if !vocab.IsNil(by) {
				// if "it" is not a public activity, save it to its actor Outbox instead of the global activities collection
				outbox := vocab.Outbox.IRI(by)
				id = outbox.AddPath(uid)
			}
			return nil
		})
		if err != nil {
			return id, err
		}
		err = vocab.OnObject(it, func(a *vocab.Object) error {
			a.ID = id
			return nil
		})
		return id, err
	}
	if it.IsLink() {
		return id, vocab.OnLink(it, func(l *vocab.Link) error {
			l.ID = id
			return nil
		})
	}
	return id, vocab.OnObject(it, func(o *vocab.Object) error {
		o.ID = id
		return nil
	})
}

func CreateService(r storage.FullStorage, self vocab.Item, keyType string) (err error) {
	_ = vocab.OnActor(self, func(service *vocab.Actor) error {
		service.Published = time.Now().UTC()
		return nil
	})
	self, err = r.Save(self)
	if err != nil {
		return err
	}

	c := osin.DefaultClient{Id: string(self.GetLink())}
	_ = r.CreateClient(&c)

	if err = AddKeyToItem(r, self, keyType); err != nil {
		return err
	}

	rr, ok := r.(storage.CollectionStore)
	if !ok {
		return nil
	}

	col := func(iri vocab.IRI) vocab.CollectionInterface {
		return &vocab.OrderedCollection{
			ID:           iri,
			Type:         vocab.OrderedCollectionType,
			Published:    time.Now().UTC(),
			AttributedTo: self.GetLink(),
			CC:           vocab.ItemCollection{vocab.PublicNS},
		}
	}
	return vocab.OnActor(self, func(service *vocab.Actor) error {
		var multi error
		for _, stream := range service.Streams {
			// NOTE(marius): create fedbox custom collections /activities, /objects, /actors
			if _, err := rr.Create(col(stream.GetID())); err != nil {
				multi = errors.Join(multi, err)
			}
		}
		return multi
	})
}
