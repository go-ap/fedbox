package activitypub

import (
	"fmt"
	"github.com/buger/jsonparser"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/auth"
	"strings"
)

// OrderedCollection should be identical to:
//    github.com/go-ap/activitystreams/collections.go#OrderedCollection
// We need it here in order to be able to implement our own UnmarshalJSON() method
type OrderedCollection as.OrderedCollection

// Collection should be identical to:
//    github.com/go-ap/activitystreams/collections.go#Collection
// We need it here in order to be able to implement our own UnmarshalJSON() method
type Collection as.Collection

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
	return nil
}

// Count returns the maximum between the length of Items in collection and its TotalItems property
func (o *OrderedCollection) Count() uint {
	if o.TotalItems > 0 {
		return o.TotalItems
	}
	return uint(len(o.OrderedItems))
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
		if as.ActivityTypes.Contains(it.GetType()) {
			act := &as.Activity{}
			if data, _, _, err := jsonparser.Get(data, "orderedItems", fmt.Sprintf("[%d]", i)); err == nil {
				act.UnmarshalJSON(data)
			}
			if context, err := jsonparser.GetString(data, "orderedItems", fmt.Sprintf("[%d]", i), "context"); err == nil {
				act.Context = as.IRI(context)
			}
			a = act
		} else if as.ObjectTypes.Contains(it.GetType()) {
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
				p := &auth.Person{}
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

// Count returns the maximum between the length of Items in collection and its TotalItems property
func (c *Collection) Count() uint {
	if c.TotalItems > 0 {
		return c.TotalItems
	}
	return uint(len(c.Items))
}

// Append adds an element to an Collection
func (c *Collection) Append(ob as.Item) error {
	c.Items = append(c.Items, ob)
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
		if as.ActivityTypes.Contains(it.GetType()) {
			act := &as.Activity{}
			if data, _, _, err := jsonparser.Get(data, "items", fmt.Sprintf("[%d]", i)); err == nil {
				act.UnmarshalJSON(data)
			}
			if context, err := jsonparser.GetString(data, "items", fmt.Sprintf("[%d]", i), "context"); err == nil {
				act.Context = as.IRI(context)
			}
			a = act
		} else if as.ObjectTypes.Contains(it.GetType()) {
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
				p := &auth.Person{}
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

// Contains verifies if Collection array contains the received one
func(c Collection) Contains(r as.IRI) bool {
	if len(c.Items) == 0 {
		return false
	}
	for _, iri := range c.Items {
		if strings.ToLower(r.String()) == strings.ToLower(iri.GetLink().String()) {
			return true
		}
	}
	return false
}

// Contains verifies if OrderedCollection array contains the received one
func(o OrderedCollection) Contains(r as.IRI) bool {
	if len(o.OrderedItems) == 0 {
		return false
	}
	for _, iri := range o.OrderedItems {
		if strings.ToLower(r.String()) == strings.ToLower(iri.GetLink().String()) {
			return true
		}
	}
	return false
}
