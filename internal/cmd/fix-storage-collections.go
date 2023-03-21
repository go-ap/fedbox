package cmd

import (
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox"
	"github.com/go-ap/filters"
	"github.com/go-ap/processing"
	"github.com/urfave/cli/v2"
	"time"
)

var FixStorageCollectionsCmd = &cli.Command{
	Name:   "fix-storage",
	Usage:  "Fix storage collections helper",
	Action: fixStorageCollectionsAct(&ctl),
}

var allCollectionPaths = append(filters.FedBOXCollections, vocab.ActivityPubCollections...)

func newOrderedCollection(id vocab.IRI) *vocab.OrderedCollection {
	return &vocab.OrderedCollection{
		ID:        id,
		Type:      vocab.OrderedCollectionType,
		Generator: ctl.Service.GetLink(),
		Published: time.Now().UTC(),
	}
}

func fixStorageCollectionsAct(ctl *Control) cli.ActionFunc {
	return func(c *cli.Context) error {
		if _, ok := ctl.Storage.(processing.CollectionStore); !ok {
			return errors.Newf("Invalid storage type %T. Unable to handle collection operations.", ctl.Storage)
		}
		if ctl.Service.Streams != nil {
			for _, stream := range ctl.Service.Streams {
				if _, maybeCol := allCollectionPaths.Split(stream.GetLink()); !allCollectionPaths.Contains(maybeCol) {
					ctl.Logger.Debugf("Stream doesn't seem to be a collection", stream)
					return nil
				}
				err := tryCreateCollection(ctl.Storage, stream.GetLink())
				if err != nil {
					continue
				}
			}
		}
		return nil
	}
}

func tryCreateCollection(storage fedbox.FullStorage, colIRI vocab.IRI) error {
	var collection *vocab.OrderedCollection
	items, err := ctl.Storage.Load(colIRI.GetLink())
	if err != nil {
		if !errors.IsNotFound(err) {
			ctl.Logger.Errorf("Unable to load %s: %s", colIRI, err)
			return err
		}
		colSaver, ok := storage.(processing.CollectionStore)
		if !ok {
			return errors.Newf("Invalid storage type %T. Unable to handle collection operations.", storage)
		}
		it, err := colSaver.Create(newOrderedCollection(colIRI.GetLink()))
		if err != nil {
			ctl.Logger.Errorf("Unable to create collection %s: %s", colIRI, err)
			return err
		}
		collection, err = vocab.ToOrderedCollection(it)
		if err != nil {
			ctl.Logger.Errorf("Saved object is not a valid OrderedCollection, but %s: %s", it.GetType(), err)
			return err
		}
	}

	if vocab.IsNil(items) {
		return nil
	}

	if !items.IsCollection() {
		ctl.Logger.Infof("Try saving %s", items.GetID())
		if _, err := storage.Save(items); err != nil {
			ctl.Logger.Errorf("Unable to save object %s: %s", items.GetLink(), err)
			return err
		}
	}
	collection, err = vocab.ToOrderedCollection(items)
	if err != nil {
		ctl.Logger.Errorf("Saved object is not a valid OrderedCollection, but %s: %s", items.GetType(), err)
		return err
	}
	vocab.OnCollectionIntf(items, func(col vocab.CollectionInterface) error {
		collection.TotalItems = col.Count()
		for _, it := range col.Collection() {
			// Try saving objects in collection, which would create the collections if they exist
			ctl.Logger.Infof("Try saving %s", it.GetID())
			if _, err := storage.Save(it); err != nil {
				ctl.Logger.Errorf("Unable to save object %s: %s", it.GetLink(), err)
			}
		}
		return nil
	})

	collection.OrderedItems = nil
	_, err = storage.Save(collection)
	if err != nil {
		ctl.Logger.Errorf("Unable to save collection with updated totalItems", err)
		return err
	}

	return nil
}
