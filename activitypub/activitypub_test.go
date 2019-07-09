package activitypub

import (
	"github.com/go-ap/activitystreams"
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
	t.Skipf("TODO")
}
