package activitypub

import (
	"github.com/go-ap/activitypub"
	as "github.com/go-ap/activitystreams"
	"reflect"
	"testing"
	"time"
)

func TestFilters_Count(t *testing.T) {
	t.Skipf("TODO")
}

func TestFilters_GetLink(t *testing.T) {
	val := as.IRI("http://example.com")
	f := Filters{
		IRI: val,
	}

	if f.GetLink() != val {
		t.Errorf("Invalid Link returned %s, expected %s", f.GetLink(), val)
	}
}

func TestFilters_IRIs(t *testing.T) {
	val := "http://example.com"
	val1 := "http://example1.com"
	val2 := "http://example1.com/test"
	f := Filters{
		ItemKey: []Hash{Hash(val), Hash(val1), Hash(val2)},
	}
	fullIris := as.IRIs{
		as.IRI(val),
		as.IRI(val1),
		as.IRI(val2),
	}

	if !f.IRIs().Contains(as.IRI(val)) {
		t.Errorf("Invalid IRIs returned %v, expected %s", f.IRIs(), val)
	}
	if !f.IRIs().Contains(as.IRI(val1)) {
		t.Errorf("Invalid IRIs returned %v, expected %s", f.IRIs(), val1)
	}
	if !f.IRIs().Contains(as.IRI(val2)) {
		t.Errorf("Invalid IRIs returned %v, expected %s", f.IRIs(), val2)
	}
	if !reflect.DeepEqual(f.IRIs(), fullIris) {
		t.Errorf("Invalid IRIs returned %v, expected %s", f.IRIs(), fullIris)
	}

}

func TestFilters_Page(t *testing.T) {
	t.Skipf("TODO")
}

func TestFilters_Types(t *testing.T) {
	t.Skipf("TODO")
}

func TestFromRequest(t *testing.T) {
	t.Skipf("TODO")
}

func TestHash_String(t *testing.T) {
	t.Skipf("TODO")
}

func TestValidActivityCollection(t *testing.T) {
	t.Skipf("TODO")
}

func mockItem() activitypub.Object {
	return activitypub.Object{
		Parent: as.Parent{
			ID:           "",
			Type:         "",
			Name:         nil,
			Attachment:   nil,
			AttributedTo: nil,
			Audience:     nil,
			Content:      nil,
			Context:      nil,
			MediaType:    "",
			EndTime:      time.Time{},
			Generator:    nil,
			Icon:         nil,
			Image:        nil,
			InReplyTo:    nil,
			Location:     nil,
			Preview:      nil,
			Published:    time.Time{},
			Replies:      nil,
			StartTime:    time.Time{},
			Summary:      nil,
			Tag:          nil,
			Updated:      time.Time{},
			URL:          nil,
			To:           nil,
			Bto:          nil,
			CC:           nil,
			BCC:          nil,
			Duration:     0,
		},
		Likes:  nil,
		Shares: nil,
		Source: activitypub.Source{},
	}
}

func TestFilters_Actors(t *testing.T) {
	f := Filters{
		ActorKey: []Hash{Hash("test")},
	}

	if f.Actors() == nil {
		t.Errorf("Actors() should not return nil")
		return
	}
	act := mockActivity()
	act.Actor = as.IRI("/actors/test")
	t.Run("exists", func(t *testing.T) {
		if !testItInIRIs(f.Actors(), act.Actor) {
			t.Errorf("filter %v doesn't contain any of %v", f.Objects(), act.Actor)
		}
	})
	act.Actor = as.ItemCollection{as.IRI("/actors/test123"), as.IRI("https://example.com")}
	t.Run("missing", func(t *testing.T) {
		if testItInIRIs(f.Actors(), act.Actor) {
			t.Errorf("filter %v shouldn't contain any of %v", f.Objects(), act.Actor)
		}
	})
}

func testItInIRIs(iris as.IRIs, items ...as.Item) bool {
	contains := false
	for _, val := range items {
		if val.IsCollection() {
			activitypub.OnCollection(val, func(c as.CollectionInterface) error {
				for _, it := range c.Collection() {
					if iris.Contains(it.GetLink()) {
						contains = true
						return nil
					}
				}
				return nil
			})
		}
		if iris.Contains(val.GetLink()) {
			contains = true
			break
		}
	}
	return contains
}

func TestFilters_AttributedTo(t *testing.T) {
	f := Filters{
		InReplTo: []Hash{"test"},
	}

	if f.InReplyTo() == nil {
		t.Errorf("InReplyTo() should not return nil")
		return
	}
	it := mockItem()
	it.InReplyTo = as.ItemCollection{as.IRI("/objects/test")}
	t.Run("exists", func(t *testing.T) {
		if !testItInIRIs(f.InReplyTo(), it.InReplyTo) {
			t.Errorf("filter %v doesn't contain any of %v", f.InReplyTo(), it.InReplyTo)
		}
	})
	it.InReplyTo = as.ItemCollection{as.IRI("/objects/test123"), as.IRI("https://example.com")}
	t.Run("missing", func(t *testing.T) {
		if testItInIRIs(f.InReplyTo(), it.InReplyTo) {
			t.Errorf("filter %v shouldn't contain any of %v", f.InReplyTo(), it.InReplyTo)
		}
	})
}

func TestFilters_Audience(t *testing.T) {
	f := Filters{
		Aud: as.IRIs{as.IRI("test")},
	}
	if f.Audience() == nil {
		t.Errorf("Audience() should not return nil")
		return
	}
	it := mockItem()
	it.Audience = as.ItemCollection{as.IRI("/actors/test")}
	t.Run("exists", func(t *testing.T) {
		if !testItInIRIs(f.Audience(), it.Audience...) {
			t.Errorf("filter %v doesn't contain any of %v", f.Audience(), it.Audience)
		}
	})
	it.Audience = as.ItemCollection{as.IRI("/actors/test123"), as.IRI("https://example.com")}
	t.Run("missing", func(t *testing.T) {
		if testItInIRIs(f.Audience(), it.Audience...) {
			t.Errorf("filter %v shouldn't contain any of %v", f.Audience(), it.Audience)
		}
	})
}

func TestFilters_Context(t *testing.T) {
	f := Filters{
		OP: []Hash{Hash("test")},
	}
	if f.Context() == nil {
		t.Errorf("Context() should not return nil")
		return
	}
	it := mockItem()
	it.Context = as.IRI("/objects/test")
	t.Run("exists", func(t *testing.T) {
		if !testItInIRIs(f.Context(), it.Context) {
			t.Errorf("filter %v doesn't contain any of %v", f.Context(), it.Context)
		}
	})
	it.Context = as.ItemCollection{as.IRI("/objects/test123"), as.IRI("https://example.com")}
	t.Run("missing", func(t *testing.T) {
		if testItInIRIs(f.Context(), it.Context) {
			t.Errorf("filter %v shouldn't contain any of %v", f.Context(), it.Context)
		}
	})
}

func TestFilters_InReplyTo(t *testing.T) {
	f := Filters{
		InReplTo: []Hash{Hash("test")},
	}
	if f.InReplyTo() == nil {
		t.Errorf("InReplyTo() should not return nil")
		return
	}
	it := mockItem()
	it.InReplyTo = as.ItemCollection{as.IRI("/objects/test")}
	t.Run("exists", func(t *testing.T) {
		if !testItInIRIs(f.InReplyTo(), it.InReplyTo) {
			t.Errorf("filter %v doesn't contain any of %v", f.InReplyTo(), it.InReplyTo)
		}
	})
	it.InReplyTo = as.ItemCollection{as.IRI("/objects/test123"), as.IRI("https://example.com")}
	t.Run("missing", func(t *testing.T) {
		if testItInIRIs(f.InReplyTo(), it.InReplyTo) {
			t.Errorf("filter %v shouldn't contain any of %v", f.InReplyTo(), it.InReplyTo)
		}
	})
}

func TestFilters_MediaTypes(t *testing.T) {
	t.Skipf("TODO")
}

func TestFilters_Names(t *testing.T) {
	t.Skipf("TODO")
}

func mockActivity() as.Activity {
	return as.Activity{
		Parent: as.Parent{
			ID:           "",
			Type:         "",
			Name:         nil,
			Attachment:   nil,
			AttributedTo: nil,
			Audience:     nil,
			Content:      nil,
			Context:      nil,
			MediaType:    "",
			EndTime:      time.Time{},
			Generator:    nil,
			Icon:         nil,
			Image:        nil,
			InReplyTo:    nil,
			Location:     nil,
			Preview:      nil,
			Published:    time.Time{},
			Replies:      nil,
			StartTime:    time.Time{},
			Summary:      nil,
			Tag:          nil,
			Updated:      time.Time{},
			URL:          nil,
			To:           nil,
			Bto:          nil,
			CC:           nil,
			BCC:          nil,
			Duration:     0,
		},
		Actor:      nil,
		Target:     nil,
		Result:     nil,
		Origin:     nil,
		Instrument: nil,
		Object:     nil,
	}

}
func TestFilters_Objects(t *testing.T) {
	f := Filters{
		ObjectKey: []Hash{Hash("test")},
	}
	if f.Objects() == nil {
		t.Errorf("Object() should not return nil")
		return
	}
	act := mockActivity()
	act.Object = as.IRI("/objects/test")
	t.Run("exists", func(t *testing.T) {
		if !testItInIRIs(f.Objects(), act.Object) {
			t.Errorf("filter %v doesn't contain any of %v", f.Objects(), act.Object)
		}
	})
	act.Object = as.ItemCollection{as.IRI("/objects/test123"), as.IRI("https://example.com")}
	t.Run("missing", func(t *testing.T) {
		if testItInIRIs(f.Objects(), act.Object) {
			t.Errorf("filter %v shouldn't contain any of %v", f.Objects(), act.Object)
		}
	})
}

func TestFilters_Targets(t *testing.T) {
	f := Filters{
		TargetKey: []Hash{Hash("test")},
	}
	act := mockActivity()
	act.Target = as.IRI("/objects/test")
	t.Run("exists", func(t *testing.T) {
		if !testItInIRIs(f.Targets(), act.Target) {
			t.Errorf("filter %v doesn't contain any of %v", f.Targets(), act.Target)
		}
	})
	act.Target = as.ItemCollection{as.IRI("/objects/test123"), as.IRI("https://example.com")}
	t.Run("missing", func(t *testing.T) {
		if testItInIRIs(f.Targets(), act.Target) {
			t.Errorf("filter %v shouldn't contain any of %v", f.Targets(), act.Target)
		}
	})
}
func TestFilters_URLs(t *testing.T) {
	t.Skipf("TODO")
}

func TestFilters_ItemMatches(t *testing.T) {
	t.Skipf("TODO")
}

func TestFilters_FilterCollection(t *testing.T) {
	t.Skipf("TODO")
}
