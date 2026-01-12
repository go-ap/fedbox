package ap

import (
	"git.sr.ht/~mariusor/storage-all"
	vocab "github.com/go-ap/activitypub"
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

func KeyGenerator(metaSaver storage.MetadataStorage, typ KeyType) func(act *vocab.Actor) error {
	return func(act *vocab.Actor) error {
		pair, err := GenerateKeyPair(typ)
		if err != nil {
			return err
		}
		return AddKeyToPerson(metaSaver, *pair)(act)
	}
}
