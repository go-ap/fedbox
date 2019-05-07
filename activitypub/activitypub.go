package activitypub

import (
	"fmt"
	"github.com/buger/jsonparser"
	ap "github.com/go-ap/activitypub"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/fedbox/internal/errors"
	"github.com/go-ap/jsonld"
)

const Public = "https://www.w3.org/ns/activitystreams#Public"

// PublicKey holds the ActivityPub compatible public key data
type PublicKey struct {
	ID           as.ObjectID     `jsonld:"id,omitempty"`
	Owner        as.ObjectOrLink `jsonld:"owner,omitempty"`
	PublicKeyPem string          `jsonld:"publicKeyPem,omitempty"`
}

// Person it should be identical to:
//    github.com/go-ap/activitypub/actors.go#Actor
// We need it here in order to be able to add to it our Score property
type Person struct {
	ap.Person
	PublicKey PublicKey `jsonld:"publicKey,omitempty"`
}

type Service = Person
type Group = Person
type Application = Person

// OrderedCollection should be identical to:
//    github.com/go-ap/activitystreams/collections.go#OrderedCollection
// We need it here in order to be able to implement our own UnmarshalJSON() method
type OrderedCollection as.OrderedCollection

// Collection should be identical to:
//    github.com/go-ap/activitystreams/collections.go#Collection
// We need it here in order to be able to implement our own UnmarshalJSON() method
type Collection as.Collection

// Activity it should be identical to:
//    github.com/go-ap/activitystreams/activity.go#Activity
// We need it here in order to be able to implement our own UnmarshalJSON() method
type Activity as.Activity

// GetID returns the ObjectID pointer of current Person instance
func (p Person) GetID() *as.ObjectID {
	id := as.ObjectID(p.ID)
	return &id
}
func (p Person) GetType() as.ActivityVocabularyType {
	return as.ActivityVocabularyType(p.Type)
}
func (p Person) GetLink() as.IRI {
	return as.IRI(p.ID)
}
func (p Person) IsLink() bool {
	return false
}

func (p Person) IsObject() bool {
	return true
}

func (p *PublicKey) UnmarshalJSON(data []byte) error {
	if id, err := jsonparser.GetString(data, "id"); err == nil {
		p.ID = as.ObjectID(id)
	} else {
		return err
	}
	if o, err := jsonparser.GetString(data, "owner"); err == nil {
		p.Owner = as.IRI(o)
	} else {
		return err
	}
	if pub, err := jsonparser.GetString(data, "publicKeyPem"); err == nil {
		p.PublicKeyPem = pub
	} else {
		return err
	}
	return nil
}

// UnmarshalJSON tries to load json data to Person object
func (p *Person) UnmarshalJSON(data []byte) error {
	app := ap.Person{}
	if err := app.UnmarshalJSON(data); err != nil {
		return err
	}

	p.Person = app
	if pubData, _, _, err := jsonparser.Get(data, "publicKey"); err == nil {
		p.PublicKey.UnmarshalJSON(pubData)
	}

	return nil
}

// CollectionNew initializes a new Collection
func OrderedCollectionNew(id as.ObjectID) *OrderedCollection {
	o := OrderedCollection(*as.OrderedCollectionNew(id))
	return &o
}

// GetType returns the OrderedCollection's type
func (o OrderedCollection) GetType() as.ActivityVocabularyType {
	return o.Type
}

// GetLink returns the IRI of the OrderedCollection object
func (o OrderedCollection) GetLink() as.IRI {
	return as.IRI(o.ID)
}

// IsLink returns false for an OrderedCollection object
func (o OrderedCollection) IsLink() bool {
	return false
}

// GetID returns the ObjectID corresponding to the OrderedCollection
func (o OrderedCollection) GetID() *as.ObjectID {
	return &o.ID
}

// IsObject returns true for am OrderedCollection object
func (o OrderedCollection) IsObject() bool {
	return true
}

// Collection returns the underlying Collection type
func (o *OrderedCollection) Collection() as.CollectionInterface {
	return o
}

// Append adds an element to an OrderedCollection
func (o *OrderedCollection) Append(ob as.Item) error {
	o.OrderedItems = append(o.OrderedItems, ob)
	o.TotalItems++
	return nil
}

// UnmarshalJSON tries to load json data to OrderedCollection o
func (o *OrderedCollection) UnmarshalJSON(data []byte) error {
	col := as.OrderedCollection{}
	err := col.UnmarshalJSON(data)
	if err != nil {
		return err
	}

	var items = make(as.ItemCollection, 0)
	for i, it := range col.OrderedItems {
		var a as.ObjectOrLink
		if as.ValidActivityType(it.GetType()) {
			act := &Activity{}
			if data, _, _, err := jsonparser.Get(data, "orderedItems", fmt.Sprintf("[%d]", i)); err == nil {
				act.UnmarshalJSON(data)
			}
			if context, err := jsonparser.GetString(data, "orderedItems", fmt.Sprintf("[%d]", i), "context"); err == nil {
				act.Context = as.IRI(context)
			}
			a = act
		} else if as.ValidObjectType(it.GetType()) {
			switch it.GetType() {
			case as.ServiceType:
				fallthrough
			case as.GroupType:
				fallthrough
			case as.ApplicationType:
				fallthrough
			case as.OrganizationType:
				fallthrough
			case as.PersonType:
				p := &Person{}
				if data, _, _, err := jsonparser.Get(data, "orderedItems", fmt.Sprintf("[%d]", i)); err == nil {
					p.UnmarshalJSON(data)
				}
				if context, err := jsonparser.GetString(data, "orderedItems", fmt.Sprintf("[%d]", i), "context"); err == nil {
					p.Context = as.IRI(context)
				}
				a = p
			}
		}
		if a == nil {
			continue
		}
		items = append(items, a)
	}

	*o = OrderedCollection(col)
	o.OrderedItems = items
	o.TotalItems = uint(len(items))
	return nil
}

// CollectionNew initializes a new Collection
func CollectionNew(id as.ObjectID) *Collection {
	o := Collection(*as.CollectionNew(id))
	return &o
}

// GetType returns the Collection's type
func (c Collection) GetType() as.ActivityVocabularyType {
	return c.Type
}

// GetLink returns the IRI of the Collection object
func (c Collection) GetLink() as.IRI {
	return as.IRI(c.ID)
}

// IsLink returns false for an Collection object
func (c Collection) IsLink() bool {
	return false
}

// GetID returns the ObjectID corresponding to the Collection
func (c Collection) GetID() *as.ObjectID {
	return &c.ID
}

// IsObject returns true for am Collection object
func (c Collection) IsObject() bool {
	return true
}

// Collection returns the underlying Collection type
func (c *Collection) Collection() as.CollectionInterface {
	return c
}

// Append adds an element to an Collection
func (c *Collection) Append(ob as.Item) error {
	c.Items = append(c.Items, ob)
	c.TotalItems++
	return nil
}

// UnmarshalJSON tries to load json data to Collection c
func (c *Collection) UnmarshalJSON(data []byte) error {
	col := as.Collection{}
	err := col.UnmarshalJSON(data)
	if err != nil {
		return err
	}

	var items = make(as.ItemCollection, 0)
	for i, it := range col.Items {
		var a as.ObjectOrLink
		if as.ValidActivityType(it.GetType()) {
			act := &Activity{}
			if data, _, _, err := jsonparser.Get(data, "items", fmt.Sprintf("[%d]", i)); err == nil {
				act.UnmarshalJSON(data)
			}
			if context, err := jsonparser.GetString(data, "items", fmt.Sprintf("[%d]", i), "context"); err == nil {
				act.Context = as.IRI(context)
			}
			a = act
		} else if as.ValidObjectType(it.GetType()) {
			switch it.GetType() {
			case as.ServiceType:
				fallthrough
			case as.GroupType:
				fallthrough
			case as.ApplicationType:
				fallthrough
			case as.OrganizationType:
				fallthrough
			case as.PersonType:
				p := &Person{}
				if data, _, _, err := jsonparser.Get(data, "items", fmt.Sprintf("[%d]", i)); err == nil {
					p.UnmarshalJSON(data)
				}
				if context, err := jsonparser.GetString(data, "items", fmt.Sprintf("[%d]", i), "context"); err == nil {
					p.Context = as.IRI(context)
				}
				a = p
			}
		}
		if a == nil {
			continue
		}
		items = append(items, a)
	}

	*c = Collection(col)
	c.Items = items
	c.TotalItems = uint(len(items))
	return nil
}

// GetID returns the ObjectID pointer of current Activity instance
func (a Activity) GetID() *as.ObjectID {
	id := as.ObjectID(a.ID)
	return &id
}

// GetLink returns the IRI of the Activity object
func (a Activity) GetLink() as.IRI {
	return as.IRI(a.ID)
}

// GetType returns the current Activity's type
func (a Activity) GetType() as.ActivityVocabularyType {
	return as.ActivityVocabularyType(a.Type)
}

// IsLink returns false for an Activity object
func (a Activity) IsLink() bool {
	return false
}

// IsObject returns true for an Activity object
func (a Activity) IsObject() bool {
	return true
}

// UnmarshalJSON
func (a *Activity) UnmarshalJSON(data []byte) error {
	it := as.Activity{}
	err := it.UnmarshalJSON(data)
	if err != nil {
		return err
	}
	*a = Activity(it)
	return nil
}

// UnmarshalJSON tries to detect the type of the object in the json data and then outputs a matching
// ActivityStreams object, if possible
func UnmarshalJSON(data []byte) (as.Item, error) {
	i, err := as.UnmarshalJSON(data)
	switch i.GetType() {
	case as.ApplicationType:
		fallthrough
	case as.ServiceType:
		fallthrough
	case as.GroupType:
		fallthrough
	case as.PersonType:
		i = Person{}
		jsonld.Unmarshal(data, &i)
	case as.CollectionType:
		i = Collection{}
		jsonld.Unmarshal(data, &i)
	case as.OrderedCollectionType:
		i = OrderedCollection{}
		jsonld.Unmarshal(data, &i)
	case as.ActivityType:
		i = Activity{}
		jsonld.Unmarshal(data, &i)
	}
	return i, err
}

func (a *Activity) RecipientsDeduplication() {
	b := as.Activity(*a)
	b.RecipientsDeduplication()
}

func JSONGetItemByType(typ as.ActivityVocabularyType) (as.Item, error) {
	var ret as.Item
	var err error

	if as.ValidActorType(typ) {
		ret = &Person{}
		o := ret.(*Person)
		o.Type = typ
	} else if typ == as.CollectionType {
		ret = &Collection{}
		o := ret.(*Collection)
		o.Type = typ
	} else if typ == as.OrderedCollectionType {
		ret = &OrderedCollection{}
		o := ret.(*OrderedCollection)
		o.Type = typ
	} else {
		return as.JSONGetItemByType(typ)
	}
	return ret, err
}

func ToPerson(it as.Item) (*Person, error) {
	switch i := it.(type) {
	case *Person:
		return i, nil
	case Person:
		return &i, nil
	default:
		ob, err := as.ToObject(it)
		if err != nil {
			return nil, err
		}
		p := Person{}
		p.Person.Parent = *ob
		return &p, nil
	}
	return nil, errors.Newf("unable to convert person")
}
