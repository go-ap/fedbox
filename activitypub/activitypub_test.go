package activitypub

import (
	"reflect"
	"strings"
	"testing"

	vocab "github.com/go-ap/activitypub"
)

func TestItemByType(t *testing.T) {
	type testPairs map[vocab.ActivityVocabularyType]reflect.Type

	var collectionPtrType = reflect.TypeOf(new(*vocab.Collection)).Elem()
	var orderedCollectionPtrType = reflect.TypeOf(new(*vocab.OrderedCollection)).Elem()

	var tests = testPairs{
		vocab.CollectionType:        collectionPtrType,
		vocab.OrderedCollectionType: orderedCollectionPtrType,
	}

	for typ, test := range tests {
		t.Run(string(typ), func(t *testing.T) {
			v, err := vocab.GetItemByType(typ)
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
	var generateIDTests vocab.ActivityVocabularyTypes
	generateIDTests = append(generateIDTests, vocab.ObjectTypes...)
	generateIDTests = append(generateIDTests, vocab.ActivityTypes...)
	generateIDTests = append(generateIDTests, vocab.ActorTypes...)
	partOf := vocab.IRI("http://example.com")
	for _, typ := range generateIDTests {
		it, err := vocab.GetItemByType(typ)
		if err != nil {
			t.Errorf("Unable to create object from type: %s", err)
		}
		id, err := GenerateID(it, partOf, nil)
		if err != nil {
			t.Errorf("GenerateID failed: %s", err)
		}
		if !strings.Contains(string(id), partOf.String()) {
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
	s := Self(vocab.IRI(testURL))

	if s.ID != vocab.ID(testURL) {
		t.Errorf("Invalid ID %s, expected %s", s.ID, testURL)
	}
	if s.Type != vocab.ServiceType {
		t.Errorf("Invalid Type %s, expected %s", s.Type, vocab.ServiceType)
	}
	if !s.Name.First().Value.Equals(vocab.Content("self")) {
		t.Errorf("Invalid Name %s, expected %s", s.Name, "self")
	}
	if s.AttributedTo.GetLink() != "https://github.com/mariusor" {
		t.Errorf("Invalid AttributedTo %s, expected %s", s.AttributedTo, "https://github.com/mariusor")
	}
	if s.Audience.First().GetLink() != vocab.PublicNS {
		t.Errorf("Invalid Audience %s, expected %s", s.Audience.First(), vocab.PublicNS)
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
	if !s.Summary.First().Value.Equals(vocab.Content("Generic ActivityPub service")) {
		t.Errorf("Invalid Summary %s, expected %v", s.Summary, "Generic ActivityPub service")
	}
	if s.Tag != nil {
		t.Errorf("Invalid Tag %s, expected %v", s.Tag, nil)
	}
	testIRI := vocab.IRI(testURL)
	if s.URL != testIRI {
		t.Errorf("Invalid URL %s, expected %v", s.URL, testURL)
	}
	inb := vocab.Inbox.IRI(testIRI)
	if s.Inbox != inb {
		t.Errorf("Invalid Inbox %s, expected %v", s.Inbox, inb)
	}
}

func Test_CacheKey(t *testing.T) {
	type args struct {
		f *Filters
	}
	tests := []struct {
		name string
		args args
		want vocab.IRI
	}{
		{
			name: "example.com",
			args: args{f: &Filters{IRI: "http://example.com"}},
			want: vocab.IRI("http://example.com"),
		},
		{
			name: "authenticated",
			args: args{f: &Filters{IRI: "http://example.com", Authenticated: &vocab.Actor{ID: "http://example.com/jdoe"}}},
			want: vocab.IRI("http://jdoe@example.com"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CacheKey(tt.args.f); got != tt.want {
				t.Errorf("FiltersKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
