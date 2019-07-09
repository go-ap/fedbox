package activitypub

import (
	"fmt"
	"github.com/go-ap/activitystreams"
	"github.com/go-ap/auth"
	"github.com/go-ap/handlers"
	"reflect"
	"strings"
	"testing"
)

func TestItemByType(t *testing.T) {
	type testPairs map[activitystreams.ActivityVocabularyType]reflect.Type

	var collectionPtrType = reflect.TypeOf(new(*Collection)).Elem()
	var orderedCollectionPtrType = reflect.TypeOf(new(*OrderedCollection)).Elem()

	var tests = testPairs{
		activitystreams.CollectionType:        collectionPtrType,
		activitystreams.OrderedCollectionType: orderedCollectionPtrType,
	}

	for typ, test := range tests {
		t.Run(string(typ), func(t *testing.T) {
			v, err := ItemByType(typ)
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
	var generateIDTests activitystreams.ActivityVocabularyTypes
	generateIDTests = append(generateIDTests, activitystreams.ObjectTypes...)
	generateIDTests = append(generateIDTests, activitystreams.ActivityTypes...)
	generateIDTests = append(generateIDTests, activitystreams.ActorTypes...)
	partOf := "http://example.com"
	for _, typ := range generateIDTests {
		it, err := ItemByType(typ)
		if err != nil {
			t.Errorf("Unable to create object from type: %s", err)
		}
		id, err := GenerateID(it, partOf, nil)
		if err != nil {
			t.Errorf("GenerateID failed: %s", err)
		}
		if !strings.Contains(string(id), partOf) {
			t.Errorf("Invalid ObjectID: %s, does not contain base URL %s", id, partOf)
		}
		if id != *it.GetID() {
			t.Errorf("ObjectIDs don't match: %s, expected %s", *it.GetID(), id)
		}
	}
}

func TestDefaultServiceIRI(t *testing.T) {
	t.Skipf("TODO")
}

func TestSelf(t *testing.T) {
	testURL := "http://example.com:666"
	s := Self(activitystreams.IRI(testURL))

	if s.ID != activitystreams.ObjectID(testURL) {
		t.Errorf("Invalid ID %s, expected %s", s.ID, testURL)
	}
	if s.Type != activitystreams.ServiceType {
		t.Errorf("Invalid Type %s, expected %s", s.Type, activitystreams.ServiceType)
	}
	if s.Name.First().Value != "self" {
		t.Errorf("Invalid Name %s, expected %s", s.Name, "self")
	}
	if s.AttributedTo.GetLink() != "https://github.com/mariusor" {
		t.Errorf("Invalid AttributedTo %s, expected %s", s.AttributedTo, "https://github.com/mariusor")
	}
	if s.Audience.First().GetLink() != auth.ActivityStreamsPublicNS {
		t.Errorf("Invalid Audience %s, expected %s", s.Audience.First(),  auth.ActivityStreamsPublicNS)
	}
	if s.Content != nil {
		t.Errorf("Invalid Audience %s, expected %v", s.Content,  nil)
	}
	if s.Icon != nil {
		t.Errorf("Invalid Icon %s, expected %v", s.Icon,  nil)
	}
	if s.Image != nil {
		t.Errorf("Invalid Image %s, expected %v", s.Image,  nil)
	}
	if s.Location != nil {
		t.Errorf("Invalid Location %s, expected %v", s.Location,  nil)
	}
	if s.Summary.First().Value != "Generic ActivityPub service" {
		t.Errorf("Invalid Summary %s, expected %v", s.Summary,  "Generic ActivityPub service")
	}
	if s.Tag != nil {
		t.Errorf("Invalid Tag %s, expected %v", s.Tag,  nil)
	}
	if s.URL != activitystreams.IRI(testURL) {
		t.Errorf("Invalid URL %s, expected %v", s.URL,  testURL)
	}
	inb := activitystreams.IRI(fmt.Sprintf("%s/%s", testURL, handlers.Inbox))
	if s.Inbox != inb {
		t.Errorf("Invalid Inbox %s, expected %v", s.Inbox,  inb)
	}
	outb := activitystreams.IRI(fmt.Sprintf("%s/%s", testURL, handlers.Outbox))
	if s.Outbox != outb {
		t.Errorf("Invalid Outbox %s, expected %v", s.Outbox,  outb)
	}
}
