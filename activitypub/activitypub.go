package activitypub

import (
	"fmt"
	"github.com/buger/jsonparser"
	ap "github.com/go-ap/activitypub"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/errors"
	"github.com/go-ap/handlers"
	"github.com/pborman/uuid"
	"net/url"
	"strings"
)

var ServiceIRI as.IRI

const ActivityStreamsPublicNS = as.IRI("https://www.w3.org/ns/activitystreams#Public")

var AnonymousActor = Person{
	Person: ap.Person{
		Parent: as.Parent{
			ID:   as.ObjectID(ActivityStreamsPublicNS),
			Type: as.PersonType,
		},
	},
}

func Self(baseURL as.IRI) Service {
	return Service{
		Person: ap.Person{
			Parent: as.Person{
				ID:           as.ObjectID(baseURL),
				Type:         as.ServiceType,
				Name:         as.NaturalLanguageValues{{Ref: as.NilLangRef, Value: "self"}},
				AttributedTo: as.IRI("https://github.com/mariusor"),
				Audience: as.ItemCollection{ ActivityStreamsPublicNS },
				Content:  nil, //as.NaturalLanguageValues{{Ref: as.NilLangRef, Value: ""}},
				Icon:     nil,
				Image:    nil,
				Location: nil,
				Summary:  as.NaturalLanguageValues{{Ref: as.NilLangRef, Value: "Generic ActivityPub service"}},
				Tag:      nil,
				URL:      baseURL,
			},
			Inbox:  as.IRI(fmt.Sprintf("%s%s", baseURL, handlers.Inbox)),
			Outbox: as.IRI(fmt.Sprintf("%s%s", baseURL, handlers.Outbox)),
		},
	}
}

func DefaultServiceIRI(baseURL string) as.IRI {
	u, _ := url.Parse(baseURL)
	// TODO(marius): I don't like adding the / folder to something like http://fedbox.git
	// I need to find an
	if u.Path == "" {
		u.Path = "/"
	}
	return as.IRI(u.String())
}

// PublicKey holds the ActivityPub compatible public key data
type PublicKey struct {
	ID           as.ObjectID     `jsonld:"id,omitempty"`
	Owner        as.ObjectOrLink `jsonld:"owner,omitempty"`
	PublicKeyPem string          `jsonld:"publicKeyPem,omitempty"`
}

// Person it should be identical to:
//    github.com/go-ap/activitypub/actors.go#To
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

// ItemByType
func ItemByType(typ as.ActivityVocabularyType) (as.Item, error) {
	var ret as.Item

	if as.ActorTypes.Contains(typ) {
		o := &Person{}
		o.Type = typ
		ret = o
	} else if as.ActivityTypes.Contains(typ) {
		o := &as.Activity{}
		o.Type = typ
		ret = o
	} else if typ == as.CollectionType {
		o := &Collection{}
		o.Type = typ
		ret = o
	} else if typ == as.OrderedCollectionType {
		o := &OrderedCollection{}
		o.Type = typ
		ret = o
	} else {
		return as.JSONGetItemByType(typ)
	}
	return ret, nil
}

// ToOrderedCollection
func ToOrderedCollection(it as.Item) (*OrderedCollection, error) {
	switch o := it.(type) {
	case *OrderedCollection:
		return o, nil
	case OrderedCollection:
		return &o, nil
	case *as.OrderedCollection:
		col := OrderedCollection(*o)
		return &col, nil
	case as.OrderedCollection:
		col := OrderedCollection(o)
		return &col, nil
	}
	return nil, errors.Newf("invalid ordered collection")
}

// ToCollection
func ToCollection(it as.Item) (*Collection, error) {
	switch o := it.(type) {
	case *Collection:
		return o, nil
	case Collection:
		return &o, nil
	case *as.Collection:
		col := Collection(*o)
		return &col, nil
	case as.Collection:
		col := Collection(o)
		return &col, nil
	}
	return nil, errors.Newf("invalid  collection")
}

// ToObject
func ToObject(it as.Item) (*as.Object, error) {
	switch o := it.(type) {
	case *Person:
		return &o.Parent, nil
	case Person:
		return &o.Parent, nil
	}
	return as.ToObject(it)
}

// ToPerson
func ToPerson(it as.Item) (*Person, error) {
	switch o := it.(type) {
	case *Person:
		return o, nil
	case Person:
		return &o, nil
	}
	ob, err := ap.ToPerson(it)
	if err != nil {
		return nil, err
	}
	p := Person{}
	p.Person = *ob
	return &p, nil
}

// FlattenObjectProperties flattens the Object's properties from Object types to IRI
func FlattenPersonProperties(o *Person) *Person {
	o.Parent = *as.FlattenObjectProperties(&o.Parent)
	return o
}

// FlattenProperties flattens the Item's properties from Object types to IRI
func FlattenProperties(it as.Item) as.Item {
	if as.ActivityTypes.Contains(it.GetType()) {
		a, err := as.ToActivity(it)
		if err == nil {
			return as.FlattenActivityProperties(a)
		}
	}
	if as.ActorTypes.Contains(it.GetType()) {
		ob, err := ToPerson(it)
		if err == nil {
			return FlattenPersonProperties(ob)
		}
	}
	if it.GetType() == as.TombstoneType {
		t, err := as.ToTombstone(it)
		if err == nil {
			t.Parent = *as.FlattenObjectProperties(&t.Parent)
			return t
		}
	}
	return as.FlattenProperties(it)
}

func UpdatePersonProperties(old, new *Person) (*Person, error) {
	o, err := UpdateObjectProperties(&old.Parent, &new.Parent)
	old.Parent = *o
	old.Inbox = replaceIfItem(old.Inbox, new.Inbox)
	old.Outbox = replaceIfItem(old.Outbox, new.Outbox)
	old.Following = replaceIfItem(old.Following, new.Following)
	old.Followers = replaceIfItem(old.Followers, new.Followers)
	old.Liked = replaceIfItem(old.Liked, new.Liked)
	old.PreferredUsername = replaceIfNaturalLanguageValues(old.PreferredUsername, new.PreferredUsername)
	return old, err
}

func replaceIfItem(old, new as.Item) as.Item {
	if new == nil {
		return old
	}
	return new
}

func replaceIfItemCollection(old, new as.ItemCollection) as.ItemCollection {
	if new == nil {
		return old
	}
	return new
}

func replaceIfNaturalLanguageValues(old, new as.NaturalLanguageValues) as.NaturalLanguageValues {
	if new == nil {
		return old
	}
	return new
}

// UpdateObjectProperties updates the "old" object properties with "new's"
func UpdateObjectProperties(old, new *as.Object) (*as.Object, error) {
	old.Name = replaceIfNaturalLanguageValues(old.Name, new.Name)
	old.Attachment = replaceIfItem(old.Attachment, new.Attachment)
	old.AttributedTo = replaceIfItem(old.AttributedTo, new.AttributedTo)
	old.Audience = replaceIfItemCollection(old.Audience, new.Audience)
	old.Content = replaceIfNaturalLanguageValues(old.Content, new.Content)
	old.Context = replaceIfItem(old.Context, new.Context)
	if len(new.MediaType) > 0 {
		old.MediaType = new.MediaType
	}
	if !new.EndTime.IsZero() {
		old.EndTime = new.EndTime
	}
	old.Generator = replaceIfItem(old.Generator, new.Generator)
	old.Icon = replaceIfItem(old.Icon, new.Icon)
	old.Image = replaceIfItem(old.Image, new.Image)
	old.InReplyTo = replaceIfItem(old.InReplyTo, new.InReplyTo)
	old.Location = replaceIfItem(old.Location, new.Location)
	old.Preview = replaceIfItem(old.Preview, new.Preview)
	if !new.Published.IsZero() {
		old.Published = new.Published
	}
	old.Replies = replaceIfItem(old.Replies, new.Replies)
	if !new.StartTime.IsZero() {
		old.StartTime = new.StartTime
	}
	old.Summary = replaceIfNaturalLanguageValues(old.Summary, new.Summary)
	old.Tag = replaceIfItemCollection(old.Tag, new.Tag)
	if !new.Updated.IsZero() {
		old.Updated = new.Updated
	}
	if new.URL != nil {
		old.URL = new.URL
	}
	old.To = replaceIfItemCollection(old.To, new.To)
	old.Bto = replaceIfItemCollection(old.Bto, new.Bto)
	old.CC = replaceIfItemCollection(old.CC, new.CC)
	old.BCC = replaceIfItemCollection(old.BCC, new.BCC)
	if new.Duration == 0 {
		old.Duration = new.Duration
	}
	return old, nil
}

// UpdateItemProperties delegates to the correct per type functions for copying
// properties between matching Activity Objects
func UpdateItemProperties(to, from as.Item) (as.Item, error) {
	if to == nil {
		return to, errors.Newf("Nil object to update")
	}
	if from == nil {
		return to, errors.Newf("Nil object for update")
	}
	if *to.GetID() != *from.GetID() {
		return to, errors.Newf("Object IDs don't match")
	}
	if to.GetType() != from.GetType() {
		return to, errors.Newf("Invalid object types for update")
	}
	if as.ActorTypes.Contains(to.GetType()) {
		o, err := ToPerson(to)
		if err != nil {
			return o, err
		}
		n, err := ToPerson(from)
		if err != nil {
			return o, err
		}
		return UpdatePersonProperties(o, n)
	}
	if as.ObjectTypes.Contains(to.GetType()) {
		o, err := as.ToObject(to)
		if err != nil {
			return o, err
		}
		n, err := as.ToObject(from)
		if err != nil {
			return o, err
		}
		return UpdateObjectProperties(o, n)
	}
	return to, errors.Newf("could not process objects with type %s", to.GetType())
}

// GenerateID generates an unique identifier for the it ActivityPub Object.
func GenerateID(it as.Item, partOf string, by as.Item) (as.ObjectID, error) {
	uuid := uuid.New()
	id := as.ObjectID(fmt.Sprintf("%s/%s", strings.ToLower(partOf), uuid))
	if as.ActivityTypes.Contains(it.GetType()) {
		a, err := as.ToActivity(it)
		if err != nil {
			return id, err
		}
		a.ID = id
		it = a
	}
	if as.ActorTypes.Contains(it.GetType()) {
		p, err := ToPerson(it)
		if err != nil {
			return id, err
		}
		p.ID = id
		it = p
	}
	if as.ObjectTypes.Contains(it.GetType()) {
		switch it.GetType() {
		case as.PlaceType:
			p, err := as.ToPlace(it)
			if err != nil {
				return id, err
			}
			p.ID = id
			it = p
		case as.ProfileType:
			p, err := as.ToProfile(it)
			if err != nil {
				return id, err
			}
			p.ID = id
			it = p
		case as.RelationshipType:
			p, err := as.ToRelationship(it)
			if err != nil {
				return id, err
			}
			p.ID = id
			it = p
		case as.TombstoneType:
			p, err := as.ToTombstone(it)
			if err != nil {
				return id, err
			}
			p.ID = id
			it = p
		default:
			p, err := as.ToObject(it)
			if err != nil {
				return id, err
			}
			p.ID = id
			it = p
		}
	}
	return id, nil
}
