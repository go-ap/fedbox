package activitypub

import (
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/handlers"
	"reflect"
	"strings"
	"testing"
)

func TestItemByType(t *testing.T) {
	type testPairs map[pub.ActivityVocabularyType]reflect.Type

	var collectionPtrType = reflect.TypeOf(new(*pub.Collection)).Elem()
	var orderedCollectionPtrType = reflect.TypeOf(new(*pub.OrderedCollection)).Elem()

	var tests = testPairs{
		pub.CollectionType:        collectionPtrType,
		pub.OrderedCollectionType: orderedCollectionPtrType,
	}

	for typ, test := range tests {
		t.Run(string(typ), func(t *testing.T) {
			v, err := pub.JSONGetItemByType(typ)
			if err != nil {
				t.Error(err)
			}
			if reflect.TypeOf(v) != test {
				t.Errorf("Invalid type returned %T, expected %s", v, test.String())
			}
		})
	}
}

func TestGenerateID(t *testing.T) {
	var generateIDTests pub.ActivityVocabularyTypes
	generateIDTests = append(generateIDTests, pub.ObjectTypes...)
	generateIDTests = append(generateIDTests, pub.ActivityTypes...)
	generateIDTests = append(generateIDTests, pub.ActorTypes...)
	partOf := "http://example.com"
	for _, typ := range generateIDTests {
		it, err := pub.JSONGetItemByType(typ)
		if err != nil {
			t.Errorf("Unable to create object from type: %s", err)
		}
		id, err := GenerateID(it, partOf, nil)
		if err != nil {
			t.Errorf("GenerateID failed: %s", err)
		}
		if !strings.Contains(string(id), partOf) {
			t.Errorf("Invalid ID: %s, does not contain base URL %s", id, partOf)
		}
		if id != it.GetID() {
			t.Errorf("IDs don't match: %s, expected %s", it.GetID(), id)
		}
	}
}

func TestDefaultServiceIRI(t *testing.T) {
	t.Skipf("TODO")
}

func TestSelf(t *testing.T) {
	testURL := "http://example.com:666"
	s := Self(pub.IRI(testURL))

	if s.ID != pub.ID(testURL) {
		t.Errorf("Invalid ID %s, expected %s", s.ID, testURL)
	}
	if s.Type != pub.ServiceType {
		t.Errorf("Invalid Type %s, expected %s", s.Type, pub.ServiceType)
	}
	if !s.Name.First().Value.Equals(pub.Content("self")) {
		t.Errorf("Invalid Name %s, expected %s", s.Name, "self")
	}
	if s.AttributedTo.GetLink() != "https://github.com/mariusor" {
		t.Errorf("Invalid AttributedTo %s, expected %s", s.AttributedTo, "https://github.com/mariusor")
	}
	if s.Audience.First().GetLink() != pub.PublicNS {
		t.Errorf("Invalid Audience %s, expected %s", s.Audience.First(), pub.PublicNS)
	}
	if s.Content != nil {
		t.Errorf("Invalid Audience %s, expected %v", s.Content, nil)
	}
	if s.Icon != nil {
		t.Errorf("Invalid Icon %s, expected %v", s.Icon, nil)
	}
	if s.Image != nil {
		t.Errorf("Invalid Image %s, expected %v", s.Image, nil)
	}
	if s.Location != nil {
		t.Errorf("Invalid Location %s, expected %v", s.Location, nil)
	}
	if !s.Summary.First().Value.Equals(pub.Content("Generic ActivityPub service")) {
		t.Errorf("Invalid Summary %s, expected %v", s.Summary, "Generic ActivityPub service")
	}
	if s.Tag != nil {
		t.Errorf("Invalid Tag %s, expected %v", s.Tag, nil)
	}
	testIRI := pub.IRI(testURL)
	if s.URL != testIRI {
		t.Errorf("Invalid URL %s, expected %v", s.URL, testURL)
	}
	inb := handlers.Inbox.IRI(testIRI)
	if s.Inbox != inb {
		t.Errorf("Invalid Inbox %s, expected %v", s.Inbox, inb)
	}
}
