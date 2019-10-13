package activitypub

import (
	"github.com/go-ap/activitystreams"
	"reflect"
	"testing"
)

func TestCollection_Count(t *testing.T) {
	id := activitystreams.ObjectID("test")

	c := Collection{
		Parent: activitystreams.Parent{
			ID: id,
		},
	}

	if c.TotalItems != 0 {
		t.Errorf("Empty object should have empty TotalItems, received %d", c.TotalItems)
	}
	if len(c.Items) > 0 {
		t.Errorf("Empty object should have empty Items, received %v", c.Items)
	}
	if c.Count() != uint(len(c.Items)) {
		t.Errorf("%T.Count() returned %d, expected %d", c, c.Count(), len(c.Items))
	}

	c.Append(activitystreams.IRI("test"))
	if c.TotalItems != 0 {
		t.Errorf("Empty object should have empty TotalItems, received %d", c.TotalItems)
	}
	if c.Count() != uint(len(c.Items)) {
		t.Errorf("%T.Count() returned %d, expected %d", c, c.Count(), len(c.Items))
	}
}

func TestCollection_Append(t *testing.T) {
	id := activitystreams.ObjectID("test")
	val := activitystreams.Object{ID: activitystreams.ObjectID("grrr")}

	c := Collection{
		Parent: activitystreams.Parent{
			ID: id,
		},
	}
	c.Append(val)

	if c.Count() != 1 {
		t.Errorf("Inbox collection of %q should have one element", *c.GetID())
	}
	if !reflect.DeepEqual(c.Items[0], val) {
		t.Errorf("First item in Inbox is does not match %q", val.ID)
	}
}

func TestCollection_Collection(t *testing.T) {
	id := activitystreams.ObjectID("test")

	c := Collection{
		Parent: activitystreams.Parent{
			ID: id,
		},
	}

	if c.Collection() != &c {
		t.Errorf("Collection should return itself %q", *c.GetID())
	}
}

func TestCollection_GetID(t *testing.T) {
	id := activitystreams.ObjectID("test")

	c := Collection{
		Parent: activitystreams.Parent{
			ID: id,
		},
	}

	if *c.GetID() != id {
		t.Errorf("GetID should return %q, received %q", id, *c.GetID())
	}
}

func TestCollection_GetLink(t *testing.T) {
	id := activitystreams.ObjectID("test")
	link := activitystreams.IRI(id)

	c := Collection{
		Parent: activitystreams.Parent{
			ID: id,
		},
	}

	if c.GetLink() != link {
		t.Errorf("GetLink should return %q, received %q", link, c.GetLink())
	}
}

func TestCollection_GetType(t *testing.T) {
	id := activitystreams.ObjectID("test")

	c := Collection{
		Parent: activitystreams.Parent{
			ID:   id,
			Type: activitystreams.CollectionType,
		},
	}
	if c.GetType() != activitystreams.CollectionType {
		t.Errorf("OrderedCollection Type should be %q, received %q", activitystreams.CollectionType, c.GetType())
	}
}

func TestCollection_IsLink(t *testing.T) {
	id := activitystreams.ObjectID("test")

	c := Collection{
		Parent: activitystreams.Parent{
			ID: id,
		},
	}

	if c.IsLink() != false {
		t.Errorf("OrderedCollection should not be a link, received %t", c.IsLink())
	}
}

func TestCollection_IsObject(t *testing.T) {
	id := activitystreams.ObjectID("test")

	c := Collection{
		Parent: activitystreams.Parent{
			ID: id,
		},
	}

	if c.IsObject() != true {
		t.Errorf("OrderedCollection should be an object, received %t", c.IsObject())
	}
}

func TestCollection_UnmarshalJSON(t *testing.T) {
	c := Collection{}

	dataEmpty := []byte("{}")
	c.UnmarshalJSON(dataEmpty)
	if c.ID != "" {
		t.Errorf("Unmarshalled object should have empty ID, received %q", c.ID)
	}
	if c.Type != "" {
		t.Errorf("Unmarshalled object should have empty Type, received %q", c.Type)
	}
	if c.AttributedTo != nil {
		t.Errorf("Unmarshalled object should have empty AttributedTo, received %q", c.AttributedTo)
	}
	if len(c.Name) != 0 {
		t.Errorf("Unmarshalled object should have empty Name, received %q", c.Name)
	}
	if len(c.Summary) != 0 {
		t.Errorf("Unmarshalled object should have empty Summary, received %q", c.Summary)
	}
	if len(c.Content) != 0 {
		t.Errorf("Unmarshalled object should have empty Content, received %q", c.Content)
	}
	if c.TotalItems != 0 {
		t.Errorf("Unmarshalled object should have empty TotalItems, received %d", c.TotalItems)
	}
	if len(c.Items) > 0 {
		t.Errorf("Unmarshalled object should have empty Items, received %v", c.Items)
	}
	if c.URL != nil {
		t.Errorf("Unmarshalled object should have empty URL, received %v", c.URL)
	}
	if !c.Published.IsZero() {
		t.Errorf("Unmarshalled object should have empty Published, received %q", c.Published)
	}
	if !c.StartTime.IsZero() {
		t.Errorf("Unmarshalled object should have empty StartTime, received %q", c.StartTime)
	}
	if !c.Updated.IsZero() {
		t.Errorf("Unmarshalled object should have empty Updated, received %q", c.Updated)
	}
}

func TestOrderedCollection_Count(t *testing.T) {
	id := activitystreams.ObjectID("test")

	c := OrderedCollection{
		Parent: activitystreams.Parent{
			ID: id,
		},
	}

	if c.TotalItems != 0 {
		t.Errorf("Empty object should have empty TotalItems, received %d", c.TotalItems)
	}
	if len(c.OrderedItems) > 0 {
		t.Errorf("Empty object should have empty Items, received %v", c.OrderedItems)
	}
	if c.Count() != uint(len(c.OrderedItems)) {
		t.Errorf("%T.Count() returned %d, expected %d", c, c.Count(), len(c.OrderedItems))
	}

	c.Append(activitystreams.IRI("test"))
	if c.TotalItems != 0 {
		t.Errorf("Empty object should have empty TotalItems, received %d", c.TotalItems)
	}
	if c.Count() != uint(len(c.OrderedItems)) {
		t.Errorf("%T.Count() returned %d, expected %d", c, c.Count(), len(c.OrderedItems))
	}
}

func TestOrderedCollection_Append(t *testing.T) {
	id := activitystreams.ObjectID("test")
	val := activitystreams.Object{ID: activitystreams.ObjectID("grrr")}

	c := OrderedCollection{
		Parent: activitystreams.Parent{
			ID: id,
		},
	}
	c.Append(val)

	if c.Count() != 1 {
		t.Errorf("Inbox collection of %q should have one element", *c.GetID())
	}
	if !reflect.DeepEqual(c.OrderedItems[0], val) {
		t.Errorf("First item in Inbox is does not match %q", val.ID)
	}
}

func TestOrderedCollection_Collection(t *testing.T) {
	id := activitystreams.ObjectID("test")

	c := OrderedCollection{
		Parent: activitystreams.Parent{
			ID: id,
		},
	}

	if c.Collection() != &c {
		t.Errorf("Collection should return itself %q", *c.GetID())
	}
}

func TestOrderedCollection_GetID(t *testing.T) {
	id := activitystreams.ObjectID("test")

	c := OrderedCollection{
		Parent: activitystreams.Parent{
			ID: id,
		},
	}

	if *c.GetID() != id {
		t.Errorf("GetID should return %q, received %q", id, *c.GetID())
	}
}

func TestOrderedCollection_GetLink(t *testing.T) {
	id := activitystreams.ObjectID("test")
	link := activitystreams.IRI(id)

	c := OrderedCollection{
		Parent: activitystreams.Parent{
			ID: id,
		},
	}

	if c.GetLink() != link {
		t.Errorf("GetLink should return %q, received %q", link, c.GetLink())
	}
}

func TestOrderedCollection_GetType(t *testing.T) {
	id := activitystreams.ObjectID("test")

	c := OrderedCollection{
		Parent: activitystreams.Parent{
			ID:   id,
			Type: activitystreams.OrderedCollectionType,
		},
	}
	if c.GetType() != activitystreams.OrderedCollectionType {
		t.Errorf("OrderedCollection Type should be %q, received %q", activitystreams.OrderedCollectionType, c.GetType())
	}
}

func TestOrderedCollection_IsLink(t *testing.T) {
	id := activitystreams.ObjectID("test")

	c := OrderedCollection{
		Parent: activitystreams.Parent{
			ID: id,
		},
	}

	if c.IsLink() != false {
		t.Errorf("OrderedCollection should not be a link, received %t", c.IsLink())
	}
}

func TestOrderedCollection_IsObject(t *testing.T) {
	id := activitystreams.ObjectID("test")

	c := OrderedCollection{
		Parent: activitystreams.Parent{
			ID: id,
		},
	}

	if c.IsObject() != true {
		t.Errorf("OrderedCollection should be an object, received %t", c.IsObject())
	}
}

func TestOrderedCollection_UnmarshalJSON(t *testing.T) {
	c := OrderedCollection{}

	dataEmpty := []byte("{}")
	c.UnmarshalJSON(dataEmpty)
	if c.ID != "" {
		t.Errorf("Unmarshalled object should have empty ID, received %q", c.ID)
	}
	if c.Type != "" {
		t.Errorf("Unmarshalled object should have empty Type, received %q", c.Type)
	}
	if c.AttributedTo != nil {
		t.Errorf("Unmarshalled object should have empty AttributedTo, received %q", c.AttributedTo)
	}
	if len(c.Name) != 0 {
		t.Errorf("Unmarshalled object should have empty Name, received %q", c.Name)
	}
	if len(c.Summary) != 0 {
		t.Errorf("Unmarshalled object should have empty Summary, received %q", c.Summary)
	}
	if len(c.Content) != 0 {
		t.Errorf("Unmarshalled object should have empty Content, received %q", c.Content)
	}
	if c.TotalItems != 0 {
		t.Errorf("Unmarshalled object should have empty TotalItems, received %d", c.TotalItems)
	}
	if len(c.OrderedItems) > 0 {
		t.Errorf("Unmarshalled object should have empty OrderedItems, received %v", c.OrderedItems)
	}
	if c.URL != nil {
		t.Errorf("Unmarshalled object should have empty URL, received %v", c.URL)
	}
	if !c.Published.IsZero() {
		t.Errorf("Unmarshalled object should have empty Published, received %q", c.Published)
	}
	if !c.StartTime.IsZero() {
		t.Errorf("Unmarshalled object should have empty StartTime, received %q", c.StartTime)
	}
	if !c.Updated.IsZero() {
		t.Errorf("Unmarshalled object should have empty Updated, received %q", c.Updated)
	}
}

func TestToCollection(t *testing.T) {
	t.Skipf("TODO")
}

func TestToOrderedCollection(t *testing.T) {
	t.Skipf("TODO")
}
