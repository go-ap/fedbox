package badger

import (
	"github.com/dgraph-io/badger"
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	s "github.com/go-ap/storage"
)

type repo struct {
	d     *badger.DB
	path  string
	logFn loggerFn
	errFn loggerFn
}

// Close opens the badger databasereif possible.
func (r *repo) Open() error {
	var err error
	c := badger.DefaultOptions(r.path)
	r.db, err = badger.Open(c)
	return err
}

// Close closes the badger database if possible.
func (r *repo) Close() error {
	return errors.NotImplementedf("method not implemented")
}

// Load
func (r *repo) Load(f s.Filterable) (pub.ItemCollection, uint, error) {
	return nil, 0, errors.NotImplementedf("method not implemented")
}

// LoadActivities
func (r *repo) LoadActivities(f s.Filterable) (pub.ItemCollection, uint, error) {
	return r.Load(f)
}

// LoadObjects
func (r *repo) LoadObjects(f s.Filterable) (pub.ItemCollection, uint, error) {
	return r.Load(f)
}

// LoadActors
func (r *repo) LoadActors(f s.Filterable) (pub.ItemCollection, uint, error) {
	return r.Load(f)
}

// LoadCollection
func (r *repo) LoadCollection(f s.Filterable) (pub.CollectionInterface, error) {
	return nil, errors.NotImplementedf("method not implemented")
}

func (r *repo) CreateCollection(col pub.CollectionInterface) (pub.CollectionInterface, error) {
	return nil, errors.NotImplementedf("method not implemented")
}

// SaveActivity
func (r *repo) SaveActivity(it pub.Item) (pub.Item, error) {
	return r.SaveObject(it)
}

// SaveActor
func (r *repo) SaveActor(it pub.Item) (pub.Item, error) {
	return r.SaveObject(it)
}

// SaveObject
func (r *repo) SaveObject(it pub.Item) (pub.Item, error) {
	return nil, errors.NotImplementedf("method not implemented")
}

// RemoveFromCollection
func (r *repo) RemoveFromCollection(col pub.IRI, it pub.Item) error {
	return errors.NotImplementedf("method not implemented")
}

// AddToCollection
func (r *repo) AddToCollection(col pub.IRI, it pub.Item) error {
	return errors.NotImplementedf("method not implemented")
}

// UpdateActor
func (r *repo) UpdateActor(it pub.Item) (pub.Item, error) {
	return r.UpdateObject(it)
}

// UpdateObject
func (r *repo) UpdateObject(it pub.Item) (pub.Item, error) {
	return r.SaveObject(it)
}

func (r *repo) DeleteActor(it pub.Item) (pub.Item, error) {
	return r.DeleteObject(it)
}

// DeleteObject
func (r *repo) DeleteObject(it pub.Item) (pub.Item, error) {
	return nil, errors.NotImplementedf("method not implemented")
}

// GenerateID
func (r *repo) GenerateID(it pub.Item, by pub.Item) (pub.ID, error) {
	return "", errors.NotImplementedf("method not implemented")
}

// PasswordSet
func (r *repo) PasswordSet(it pub.Item, pw []byte) error {
	return errors.NotImplementedf("method not implemented")
}

// PasswordCheck
func (r *repo) PasswordCheck(it pub.Item, pw []byte) error {
	return errors.NotImplementedf("method not implemented")
}
