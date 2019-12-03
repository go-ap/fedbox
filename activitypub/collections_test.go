package activitypub

import (
	pub "github.com/go-ap/activitypub"
	"reflect"
	"testing"
)

func TestCollection_Count(t *testing.T) {
	id := pub.ObjectID("test")

	c := Collection{
		ID: id,
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

	c.Append(pub.IRI("test"))
	if c.TotalItems != 0 {
		t.Errorf("Empty object should have empty TotalItems, received %d", c.TotalItems)
	}
	if c.Count() != uint(len(c.Items)) {
		t.Errorf("%T.Count() returned %d, expected %d", c, c.Count(), len(c.Items))
	}
}

func TestCollection_Append(t *testing.T) {
	id := pub.ObjectID("test")
	val := pub.Object{ID: pub.ObjectID("grrr")}

	c := Collection{
		ID: id,
	}
	c.Append(val)

	if c.Count() != 1 {
		t.Errorf("Inbox collection of %q should have one element", c.GetID())
	}
	if !reflect.DeepEqual(c.Items[0], val) {
		t.Errorf("First item in Inbox is does not match %q", val.ID)
	}
}

func TestCollection_Collection(t *testing.T) {
	id := pub.ObjectID("test")

	c := Collection{
		ID: id,
	}

	if !reflect.DeepEqual(c.Collection(), c.Items) {
		t.Errorf("Collection itemss should be equal %v %v", c.Collection(), c.Items)
	}
}

func TestCollection_GetID(t *testing.T) {
	id := pub.ObjectID("test")

	c := Collection{
		ID: id,
	}

	if c.GetID() != id {
		t.Errorf("GetID should return %q, received %q", id, c.GetID())
	}
}

func TestCollection_GetLink(t *testing.T) {
	id := pub.ObjectID("test")
	link := pub.IRI(id)

	c := Collection{
		ID: id,
	}

	if c.GetLink() != link {
		t.Errorf("GetLink should return %q, received %q", link, c.GetLink())
	}
}

func TestCollection_GetType(t *testing.T) {
	id := pub.ObjectID("test")

	c := Collection{
		ID:   id,
		Type: pub.CollectionType,
	}
	if c.GetType() != pub.CollectionType {
		t.Errorf("OrderedCollection Type should be %q, received %q", pub.CollectionType, c.GetType())
	}
}

func TestCollection_IsLink(t *testing.T) {
	id := pub.ObjectID("test")

	c := Collection{
		ID: id,
	}

	if c.IsLink() != false {
		t.Errorf("OrderedCollection should not be a link, received %t", c.IsLink())
	}
}

func TestCollection_IsObject(t *testing.T) {
	id := pub.ObjectID("test")

	c := Collection{
		ID: id,
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
	id := pub.ObjectID("test")

	c := OrderedCollection{
		ID: id,
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

	c.Append(pub.IRI("test"))
	if c.TotalItems != 0 {
		t.Errorf("Empty object should have empty TotalItems, received %d", c.TotalItems)
	}
	if c.Count() != uint(len(c.OrderedItems)) {
		t.Errorf("%T.Count() returned %d, expected %d", c, c.Count(), len(c.OrderedItems))
	}
}

func TestOrderedCollection_Append(t *testing.T) {
	id := pub.ObjectID("test")
	val := pub.Object{ID: pub.ObjectID("grrr")}

	c := OrderedCollection{
		ID: id,
	}
	c.Append(val)

	if c.Count() != 1 {
		t.Errorf("Inbox collection of %q should have one element", c.GetID())
	}
	if !reflect.DeepEqual(c.OrderedItems[0], val) {
		t.Errorf("First item in Inbox is does not match %q", val.ID)
	}
}

func TestOrderedCollection_Collection(t *testing.T) {
	id := pub.ObjectID("test")

	c := OrderedCollection{
		ID: id,
	}

	if !reflect.DeepEqual(c.Collection(), c.OrderedItems) {
		t.Errorf("Collection itemss should be equal %v %v", c.Collection(), c.OrderedItems)
	}
}

func TestOrderedCollection_GetID(t *testing.T) {
	id := pub.ObjectID("test")

	c := OrderedCollection{
		ID: id,
	}

	if c.GetID() != id {
		t.Errorf("GetID should return %q, received %q", id, c.GetID())
	}
}

func TestOrderedCollection_GetLink(t *testing.T) {
	id := pub.ObjectID("test")
	link := pub.IRI(id)

	c := OrderedCollection{
		ID: id,
	}

	if c.GetLink() != link {
		t.Errorf("GetLink should return %q, received %q", link, c.GetLink())
	}
}

func TestOrderedCollection_GetType(t *testing.T) {
	id := pub.ObjectID("test")

	c := OrderedCollection{
		ID:   id,
		Type: pub.OrderedCollectionType,
	}
	if c.GetType() != pub.OrderedCollectionType {
		t.Errorf("OrderedCollection Type should be %q, received %q", pub.OrderedCollectionType, c.GetType())
	}
}

func TestOrderedCollection_IsLink(t *testing.T) {
	id := pub.ObjectID("test")

	c := OrderedCollection{
		ID: id,
	}

	if c.IsLink() != false {
		t.Errorf("OrderedCollection should not be a link, received %t", c.IsLink())
	}
}

func TestOrderedCollection_IsObject(t *testing.T) {
	id := pub.ObjectID("test")

	c := OrderedCollection{
		ID: id,
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
