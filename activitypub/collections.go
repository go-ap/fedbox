package activitypub

import (
	"fmt"
	"github.com/buger/jsonparser"
	pub "github.com/go-ap/activitypub"
)

// OrderedCollection should be identical to:
//    github.com/go-ap/activitypub/collections.go#OrderedCollection
// We need it here in order to be able to implement our own UnmarshalJSON() method
type OrderedCollection pub.OrderedCollection

// Collection should be identical to:
//    github.com/go-ap/activitypub/collections.go#Collection
// We need it here in order to be able to implement our own UnmarshalJSON() method
type Collection pub.Collection

// GetType returns the OrderedCollection's type
func (o OrderedCollection) GetType() pub.ActivityVocabularyType {
	return o.Type
}

// GetLink returns the IRI of the OrderedCollection object
func (o OrderedCollection) GetLink() pub.IRI {
	return pub.IRI(o.ID)
}

// IsLink returns false for an OrderedCollection object
func (o OrderedCollection) IsLink() bool {
	return false
}

// GetID returns the ObjectID corresponding to the OrderedCollection
func (o OrderedCollection) GetID() pub.ObjectID {
	return o.ID
}

// IsObject returns true for an OrderedCollection object
func (o OrderedCollection) IsObject() bool {
	return true
}

// IsCollection returns true for an OrderedCollection object
func (o OrderedCollection) IsCollection() bool {
	return true
}

// Collection returns the underlying Collection type
func (o *OrderedCollection) Collection() pub.ItemCollection {
	return o.OrderedItems
}

// Append adds an element to an OrderedCollection
func (o *OrderedCollection) Append(ob pub.Item) error {
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
	col := pub.OrderedCollection{}
	err := col.UnmarshalJSON(data)
	if err != nil {
		return err
	}

	var items = make(pub.ItemCollection, 0)
	for i, it := range col.OrderedItems {
		var a pub.ObjectOrLink
		if pub.ActivityTypes.Contains(it.GetType()) {
			act := &pub.Activity{}
			if data, _, _, err := jsonparser.Get(data, "orderedItems", fmt.Sprintf("[%d]", i)); err == nil {
				act.UnmarshalJSON(data)
			}
			if context, err := jsonparser.GetString(data, "orderedItems", fmt.Sprintf("[%d]", i), "context"); err == nil {
				act.Context = pub.IRI(context)
			}
			a = act
		} else if pub.ObjectTypes.Contains(it.GetType()) {
			switch it.GetType() {
			case pub.ServiceType:
				fallthrough
			case pub.GroupType:
				fallthrough
			case pub.ApplicationType:
				fallthrough
			case pub.OrganizationType:
				fallthrough
			case pub.PersonType:
				p := &pub.Actor{}
				if data, _, _, err := jsonparser.Get(data, "orderedItems", fmt.Sprintf("[%d]", i)); err == nil {
					p.UnmarshalJSON(data)
				}
				if context, err := jsonparser.GetString(data, "orderedItems", fmt.Sprintf("[%d]", i), "context"); err == nil {
					p.Context = pub.IRI(context)
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
func (c Collection) GetType() pub.ActivityVocabularyType {
	return c.Type
}

// GetLink returns the IRI of the Collection object
func (c Collection) GetLink() pub.IRI {
	return pub.IRI(c.ID)
}

// IsLink returns false for a Collection object
func (c Collection) IsLink() bool {
	return false
}

// GetID returns the ObjectID corresponding to the Collection
func (c Collection) GetID() pub.ObjectID {
	return c.ID
}

// IsObject returns true for a Collection object
func (c Collection) IsObject() bool {
	return true
}

// IsCollection returns true for a Collection object
func (c Collection) IsCollection() bool {
	return true
}

// Collection returns the underlying Collection type
func (c *Collection) Collection() pub.ItemCollection {
	return c.Items
}

// Count returns the maximum between the length of Items in collection and its TotalItems property
func (c *Collection) Count() uint {
	if c.TotalItems > 0 {
		return c.TotalItems
	}
	return uint(len(c.Items))
}

// Append adds an element to an Collection
func (c *Collection) Append(ob pub.Item) error {
	c.Items = append(c.Items, ob)
	return nil
}

// UnmarshalJSON tries to load json data to Collection c
func (c *Collection) UnmarshalJSON(data []byte) error {
	col := pub.Collection{}
	err := col.UnmarshalJSON(data)
	if err != nil {
		return err
	}

	var items = make(pub.ItemCollection, 0)
	for i, it := range col.Items {
		var a pub.ObjectOrLink
		if pub.ActivityTypes.Contains(it.GetType()) {
			act := &pub.Activity{}
			if data, _, _, err := jsonparser.Get(data, "items", fmt.Sprintf("[%d]", i)); err == nil {
				act.UnmarshalJSON(data)
			}
			if context, err := jsonparser.GetString(data, "items", fmt.Sprintf("[%d]", i), "context"); err == nil {
				act.Context = pub.IRI(context)
			}
			a = act
		} else if pub.ObjectTypes.Contains(it.GetType()) {
			switch it.GetType() {
			case pub.ServiceType:
				fallthrough
			case pub.GroupType:
				fallthrough
			case pub.ApplicationType:
				fallthrough
			case pub.OrganizationType:
				fallthrough
			case pub.PersonType:
				p := &pub.Actor{}
				if data, _, _, err := jsonparser.Get(data, "items", fmt.Sprintf("[%d]", i)); err == nil {
					p.UnmarshalJSON(data)
				}
				if context, err := jsonparser.GetString(data, "items", fmt.Sprintf("[%d]", i), "context"); err == nil {
					p.Context = pub.IRI(context)
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
func (c Collection) Contains(r pub.IRI) bool {
	if len(c.Items) == 0 {
		return false
	}
	for _, iri := range c.Items {
		if r.Equals(iri.GetLink(), false) {
			return true
		}
	}
	return false
}

// Contains verifies if OrderedCollection array contains the received one
func (o OrderedCollection) Contains(r pub.IRI) bool {
	if len(o.OrderedItems) == 0 {
		return false
	}
	for _, iri := range o.OrderedItems {
		if r.Equals(iri.GetLink(), false) {
			return true
		}
	}
	return false
}
