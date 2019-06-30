package activitypub

import (
	"github.com/go-ap/activitystreams"
	"strings"
	"testing"
)

func TestOrderedCollection_Count(t *testing.T) {
	t.Skipf("TODO")
}

func TestFlattenProperties(t *testing.T) {
	t.Skipf("TODO")
}

func TestCollection_Count(t *testing.T) {
	t.Skipf("TODO")
}

func TestJSONGetItemByType(t *testing.T) {
	t.Skipf("TODO")
}

func TestCollection_Append(t *testing.T) {
	t.Skipf("TODO")
}

func TestCollection_Collection(t *testing.T) {
	t.Skipf("TODO")
}

func TestCollection_GetID(t *testing.T) {
	t.Skipf("TODO")
}

func TestCollection_GetLink(t *testing.T) {
	t.Skipf("TODO")
}

func TestToOrderedCollection(t *testing.T) {
	t.Skipf("TODO")
}

func TestCollection_GetType(t *testing.T) {
	t.Skipf("TODO")
}

func TestCollection_IsLink(t *testing.T) {
	t.Skipf("TODO")
}

func TestCollection_IsObject(t *testing.T) {
	t.Skipf("TODO")
}

func TestFlattenPersonProperties(t *testing.T) {
	t.Skipf("TODO")
}

func TestCollection_UnmarshalJSON(t *testing.T) {
	t.Skipf("TODO")
}

func TestOrderedCollection_Append(t *testing.T) {
	t.Skipf("TODO")
}

func TestUpdateItemProperties(t *testing.T) {
	t.Skipf("TODO")
}

func TestOrderedCollection_Collection(t *testing.T) {
	t.Skipf("TODO")
}

func TestOrderedCollection_GetID(t *testing.T) {
	t.Skipf("TODO")
}

func TestOrderedCollection_GetLink(t *testing.T) {
	t.Skipf("TODO")
}

func TestOrderedCollection_GetType(t *testing.T) {
	t.Skipf("TODO")
}

func TestOrderedCollection_IsLink(t *testing.T) {
	t.Skipf("TODO")
}

func TestOrderedCollection_IsObject(t *testing.T) {
	t.Skipf("TODO")
}

func TestOrderedCollection_UnmarshalJSON(t *testing.T) {
	t.Skipf("TODO")
}

func TestPerson_GetID(t *testing.T) {
	t.Skipf("TODO")
}

func TestPerson_GetLink(t *testing.T) {
	t.Skipf("TODO")
}

func TestPerson_GetType(t *testing.T) {
	t.Skipf("TODO")
}

func TestPerson_IsLink(t *testing.T) {
	t.Skipf("TODO")
}

func TestPerson_IsObject(t *testing.T) {
	t.Skipf("TODO")
}

func TestPerson_UnmarshalJSON(t *testing.T) {
	t.Skipf("TODO")
}

func TestPublicKey_UnmarshalJSON(t *testing.T) {
	t.Skipf("TODO")
}

func TestToCollection(t *testing.T) {
	t.Skipf("TODO")
}

func TestToObject(t *testing.T) {
	t.Skipf("TODO")
}

func TestToPerson(t *testing.T) {
	t.Skipf("TODO")
}

func TestUpdateObjectProperties(t *testing.T) {
	t.Skipf("TODO")
}

func TestUpdatePersonProperties(t *testing.T) {
	t.Skipf("TODO")
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
