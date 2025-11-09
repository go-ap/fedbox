package storage

import (
	"time"

	"git.sr.ht/~mariusor/lw"
	"git.sr.ht/~mariusor/storage-all"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/processing"
)

func BootstrapFn(opt config.Options, l lw.Logger) error {
	return storage.Bootstrap(opt.StorageInitFns(l)...)
}

func CleanFn(opt config.Options, l lw.Logger) error {
	return storage.Clean(opt.StorageInitFns(l)...)
}

func CreateService(opt config.Options, self vocab.Item, l lw.Logger) (err error) {
	r, err := storage.New(opt.StorageInitFns(l)...)
	if err != nil {
		return err
	}

	if err = r.Open(); err != nil {
		return err
	}
	defer r.Close()

	self, err = r.Save(self)
	if err != nil {
		return err
	}
	rr, ok := r.(processing.CollectionStore)
	if !ok {
		return nil
	}
	col := func(iri vocab.IRI) vocab.CollectionInterface {
		return &vocab.OrderedCollection{
			ID:           iri,
			Type:         vocab.OrderedCollectionType,
			Published:    time.Now().UTC(),
			AttributedTo: self,
			CC:           vocab.ItemCollection{vocab.PublicNS},
		}
	}
	return vocab.OnActor(self, func(service *vocab.Actor) error {
		var multi error
		for _, stream := range service.Streams {
			if _, err := rr.Create(col(stream.GetID())); err != nil {
				multi = errors.Join(multi, err)
			}
		}
		return multi
	})
}

type IRIChecker interface {
	IsLocalIRI(i vocab.IRI) bool
}

func IsLocalIRI(s processing.Store) processing.IRIValidator {
	if c, ok := s.(IRIChecker); ok {
		return c.IsLocalIRI
	}
	return func(i vocab.IRI) bool {
		return false
	}
}
