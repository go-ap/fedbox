package cmd

import (
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	st "github.com/go-ap/fedbox/storage"
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
var streamCollections = vocab.CollectionPaths{
	filters.ActivitiesType,
	filters.ActorsType,
	filters.ObjectsType,
}

func newOrderedCollection(id vocab.IRI) *vocab.OrderedCollection {
	return &vocab.OrderedCollection{
		ID:        id,
		Type:      vocab.OrderedCollectionType,
		Generator: ctl.Service.GetLink(),
		Published: time.Now().UTC(),
	}
}

func getObjectCollections(act vocab.Item) vocab.IRIs {
	collections := make(vocab.IRIs, 0)
	for _, col := range vocab.OfObject {
		if colIRI := col.IRI(act); colIRI != "" {
			collections = append(collections, colIRI)
		}
	}
	return collections
}

func getActorCollections(act vocab.Item) vocab.IRIs {
	collections := make(vocab.IRIs, 0)
	for _, col := range vocab.OfActor {
		if colIRI := col.IRI(act); colIRI != "" {
			collections = append(collections, colIRI)
		}
	}
	return collections
}

func fixStorageCollectionsAct(ctl *Control) cli.ActionFunc {
	return func(c *cli.Context) error {
		if _, ok := ctl.Storage.(processing.CollectionStore); !ok {
			return errors.Newf("Invalid storage type %T. Unable to handle collection operations.", ctl.Storage)
		}
		if err := tryCreateActorCollections(ctl.Service, ctl.Storage); err != nil {
			return err
		}
		// NOTE(marius): this assumes that storage contains the actors, activities, objects streams collections
		return tryCreateAllObjectsCollections(ctl.Service, ctl.Storage)
	}
}

func tryCreateAllObjectsCollections(actor vocab.Item, storage st.FullStorage) error {
	if actor == nil {
		return nil
	}

	allCollections := make(vocab.IRIs, 0)
	err := vocab.OnActor(actor, func(actor *vocab.Actor) error {
		if actor.Streams == nil {
			return nil
		}
		for _, stream := range actor.Streams {
			_, maybeCol := allCollectionPaths.Split(stream.GetLink())
			if !streamCollections.Contains(maybeCol) {
				continue
			}
			iri := maybeCol.IRI(ctl.Service)
			items, err := ctl.Storage.Load(iri)
			if err != nil {
				ctl.Logger.Debugf("Unable to load collection %s: %s", iri, err)
				continue
			}
			vocab.OnCollectionIntf(items, func(col vocab.CollectionInterface) error {
				for _, it := range col.Collection() {
					if vocab.ActorTypes.Contains(it.GetType()) {
						allCollections = append(allCollections, getActorCollections(it)...)
					} else {
						if it.IsCollection() {
							continue
						}
						allCollections = append(allCollections, getObjectCollections(it)...)
					}
				}
				return nil
			})
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, col := range allCollections {
		if err := tryCreateCollection(storage, col); err != nil {
			ctl.Logger.Warnf("Error when trying to create collection: %+s", err)
			continue
		}
	}
	return nil
}

func tryCreateActorCollections(actor vocab.Item, storage st.FullStorage) error {
	initialCollections := make([]vocab.IRI, 0)
	initialCollections = append(initialCollections, getActorCollections(actor)...)
	err := vocab.OnActor(actor, func(actor *vocab.Actor) error {
		if actor.Streams == nil {
			return nil
		}
		for _, stream := range actor.Streams {
			if _, maybeCol := allCollectionPaths.Split(stream.GetLink()); !allCollectionPaths.Contains(maybeCol) {
				ctl.Logger.Debugf("Stream doesn't seem to be a collection", stream)
				return nil
			}
			initialCollections = append(initialCollections, stream.GetLink())
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, col := range initialCollections {
		err := tryCreateCollection(storage, col)
		if err != nil {
			ctl.Logger.Warnf("Error when trying to create collection: %+s", err)
			continue
		}
	}
	return nil
}

func tryCreateCollection(storage st.FullStorage, colIRI vocab.IRI) error {
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
