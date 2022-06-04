package activitypub

import (
	"reflect"
	"testing"
	"time"

	vocab "github.com/go-ap/activitypub"
)

func TestFilters_GetLink(t *testing.T) {
	val := vocab.IRI("http://example.com")
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
		ItemKey: CompStrs{CompStr{Str: val}, CompStr{Str: val1}, CompStr{Str: val2}},
	}
	fullIris := CompStrs{
		CompStr{Str: val},
		CompStr{Str: val1},
		CompStr{Str: val2},
	}

	if !f.IRIs().Contains(CompStr{Str: val}) {
		t.Errorf("Invalid IRIs returned %v, expected %s", f.IRIs(), val)
	}
	if !f.IRIs().Contains(CompStr{Str: val1}) {
		t.Errorf("Invalid IRIs returned %v, expected %s", f.IRIs(), val1)
	}
	if !f.IRIs().Contains(CompStr{Str: val2}) {
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

func mockItem() vocab.Object {
	return vocab.Object{
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
		Likes:        nil,
		Shares:       nil,
		Source:       vocab.Source{},
	}
}

func LikeString(s string) CompStr {
	return CompStr{Operator: "~", Str: s}
}

func EqualsString(s string) CompStr {
	return CompStr{Operator: "=", Str: s}
}

func IRIsFilter(iris ...vocab.IRI) CompStrs {
	r := make(CompStrs, len(iris))
	for i, iri := range iris {
		r[i] = EqualsString(iri.String())
	}
	return r
}

func TestFilters_Actors(t *testing.T) {
	f := Filters{
		Actor: &Filters{Key: []Hash{Hash("test")}},
	}

	if f.Actors() == nil {
		t.Errorf("Actors() should not return nil")
		return
	}
	act := mockActivity()
	act.Actor = vocab.IRI("/actors/test")
	t.Run("exists", func(t *testing.T) {
		if !testItInIRIs(IRIsFilter(f.Actors()...), act.Actor) {
			t.Errorf("filter %v doesn't contain any of %v", f.Objects(), act.Actor)
		}
	})
	act.Actor = vocab.ItemCollection{vocab.IRI("/actors/test123"), vocab.IRI("https://example.com")}
	t.Run("missing", func(t *testing.T) {
		if testItInIRIs(IRIsFilter(f.Actors()...), act.Actor) {
			t.Errorf("filter %v shouldn't contain any of %v", f.Objects(), act.Actor)
		}
	})
}

func testItInIRIs(iris CompStrs, items ...vocab.Item) bool {
	contains := false
	for _, val := range items {
		if val.IsCollection() {
			vocab.OnCollectionIntf(val, func(c vocab.CollectionInterface) error {
				for _, it := range c.Collection() {
					if filterItem(iris, it) {
						contains = true
						return nil
					}
				}
				return nil
			})
		}
		if filterItemCollections(iris, val) {
			contains = true
			break
		}
	}
	return contains
}

func TestFilters_AttributedTo(t *testing.T) {
	f := Filters{
		InReplTo: CompStrs{CompStr{Str: "test"}},
	}

	if f.InReplyTo() == nil {
		t.Errorf("InReplyTo() should not return nil")
		return
	}
	it := mockItem()
	it.InReplyTo = vocab.ItemCollection{vocab.IRI("/objects/test")}
	t.Run("exists", func(t *testing.T) {
		if !testItInIRIs(f.InReplyTo(), it.InReplyTo) {
			t.Errorf("filter %v doesn't contain any of %v", f.InReplyTo(), it.InReplyTo)
		}
	})
	it.InReplyTo = vocab.ItemCollection{vocab.IRI("/objects/test123"), vocab.IRI("https://example.com")}
	t.Run("missing", func(t *testing.T) {
		if testItInIRIs(f.InReplyTo(), it.InReplyTo) {
			t.Errorf("filter %v shouldn't contain any of %v", f.InReplyTo(), it.InReplyTo)
		}
	})
}

func TestFilters_Audience(t *testing.T) {
	type args struct {
		filters CompStrs
		valArr  vocab.ItemCollection
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "basic-equality",
			args: args{
				filters: CompStrs{
					CompStr{
						Operator: "",
						Str:      "ana",
					},
				},
				valArr: vocab.ItemCollection{
					vocab.IRI("ana"),
				},
			},
			want: true,
		},
		{
			name: "basic-equality-with-nil-first",
			args: args{
				filters: CompStrs{
					CompStr{
						Operator: "",
						Str:      "ana",
					},
				},
				valArr: vocab.ItemCollection{
					nil,
					vocab.IRI("ana"),
				},
			},
			want: true,
		},
		{
			name: "basic-like",
			args: args{
				filters: CompStrs{
					CompStr{
						Operator: "~",
						Str:      "ana",
					},
				},
				valArr: vocab.ItemCollection{
					vocab.IRI("ana"),
				},
			},
			want: true,
		},
		{
			name: "basic-like-with-longer-value",
			args: args{
				filters: CompStrs{
					CompStr{
						Operator: "~",
						Str:      "ana",
					},
				},
				valArr: vocab.ItemCollection{
					vocab.IRI("anathema"),
				},
			},
			want: true,
		},
		{
			name: "basic-different",
			args: args{
				filters: CompStrs{
					CompStr{
						Operator: "!",
						Str:      "ana",
					},
				},
				valArr: vocab.ItemCollection{
					vocab.IRI("bob"),
				},
			},
			want: true,
		},
		{
			name: "basic-different-with-empty-values",
			args: args{
				filters: CompStrs{
					CompStr{
						Operator: "!",
						Str:      "ana",
					},
				},
				valArr: vocab.ItemCollection{
					nil,
					vocab.IRI(""),
					vocab.IRI("bob"),
				},
			},
			want: true,
		},
		{
			name: "basic-false-equality",
			args: args{
				filters: CompStrs{
					CompStr{
						Operator: "",
						Str:      "ana",
					},
				},
				valArr: vocab.ItemCollection{
					vocab.IRI("bob"),
				},
			},
			want: false,
		},
		{
			name: "basic-false-equality-with-nil-first",
			args: args{
				filters: CompStrs{
					CompStr{
						Operator: "",
						Str:      "ana",
					},
				},
				valArr: vocab.ItemCollection{
					nil,
					vocab.IRI("bob"),
				},
			},
			want: false,
		},
		{
			name: "basic-false-like",
			args: args{
				filters: CompStrs{
					CompStr{
						Operator: "~",
						Str:      "ana",
					},
				},
				valArr: vocab.ItemCollection{
					vocab.IRI("bob"),
				},
			},
			want: false,
		},
		{
			name: "basic-false-like-with-longer-value",
			args: args{
				filters: CompStrs{
					CompStr{
						Operator: "~",
						Str:      "ana",
					},
				},
				valArr: vocab.ItemCollection{
					vocab.IRI("bobsyouruncle"),
				},
			},
			want: false,
		},
		{
			name: "basic-false-different",
			args: args{
				filters: CompStrs{
					CompStr{
						Operator: "!",
						Str:      "ana",
					},
				},
				valArr: vocab.ItemCollection{
					vocab.IRI("ana"),
				},
			},
			want: false,
		},
		{
			name: "basic-false-different-with-empty-values",
			args: args{
				filters: CompStrs{
					CompStr{
						Operator: "!",
						Str:      "ana",
					},
				},
				valArr: vocab.ItemCollection{
					nil,
					vocab.IRI(""),
				},
			},
			want: true,
		},
		{
			name: "one value: exact match success",
			args: args{
				CompStrs{StringEquals("ana")},
				vocab.ItemCollection{
					vocab.IRI("ana"),
				},
			},
			want: true,
		},
		{
			name: "one value: exact match failure",
			args: args{
				CompStrs{StringEquals("ana")},
				vocab.ItemCollection{
					vocab.IRI("na"),
				},
			},
			want: false,
		},
		{
			name: "one value: partial match success",
			args: args{
				CompStrs{StringLike("ana")},
				vocab.ItemCollection{
					vocab.IRI("analema"),
				},
			},
			want: true,
		},
		{
			name: "one value: exact match failure",
			args: args{
				CompStrs{StringLike("ana")},
				vocab.ItemCollection{
					vocab.IRI("na"),
				},
			},
			want: false,
		},
		{
			name: "one value: negated match success",
			args: args{
				CompStrs{StringDifferent("ana")},
				vocab.ItemCollection{
					vocab.IRI("lema"),
				},
			},
			want: true,
		},
		{
			name: "one value: negated match failure",
			args: args{
				CompStrs{StringDifferent("ana")},
				vocab.ItemCollection{
					vocab.IRI("ana"),
				},
			},
			want: false,
		},
		// multiple filters
		{
			name: "multi filters: exact match success",
			args: args{
				CompStrs{StringEquals("ana")},
				vocab.ItemCollection{
					vocab.IRI("not-matching"),
					vocab.IRI("ana"),
				},
			},
			want: true,
		},
		{
			name: "multi filters: exact match failure",
			args: args{
				CompStrs{StringEquals("ana")},
				vocab.ItemCollection{
					vocab.IRI("not-matching"),
					vocab.IRI("na"),
				},
			},
			want: false,
		},
		{
			name: "multi filters: partial match success",
			args: args{
				CompStrs{StringLike("ana")},
				vocab.ItemCollection{
					vocab.IRI("not-matching"),
					vocab.IRI("analema"),
				},
			},
			want: true,
		},
		{
			name: "multi filters: exact match failure",
			args: args{
				CompStrs{StringLike("ana")},
				vocab.ItemCollection{
					vocab.IRI("not-matching"),
					vocab.IRI("na"),
				},
			},
			want: false,
		},
		{
			name: "multi filters: negated match success",
			args: args{
				CompStrs{StringDifferent("ana")},
				vocab.ItemCollection{
					vocab.IRI("not-matching"),
					vocab.IRI("lema"),
				},
			},
			want: true,
		},
		{
			name: "multi filters: negated match failure",
			args: args{
				CompStrs{StringDifferent("ana")},
				vocab.ItemCollection{
					vocab.IRI("not-matching"),
					vocab.IRI("ana"),
				},
			},
			want: false,
		},
		{
			name: "existing_matching",
			args: args{
				filters: CompStrs{CompStr{Str: "/actors/test"}},
				valArr:  vocab.ItemCollection{vocab.IRI("/actors/test")},
			},
			want: true,
		},
		{
			name: "existing_not_matching",
			args: args{
				filters: CompStrs{CompStr{Str: "/actors/test"}},
				valArr:  vocab.ItemCollection{vocab.IRI("/actors/test123"), vocab.IRI("https://example.com")},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filterAudience(tt.args.filters, tt.args.valArr); got != tt.want {
				t.Errorf("filterAudience() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilters_Context(t *testing.T) {
	f := Filters{
		OP: CompStrs{EqualsString("test")},
	}
	if f.Context() == nil {
		t.Errorf("Context() should not return nil")
		return
	}
	it := mockItem()
	it.Context = vocab.IRI("/objects/test")
	t.Run("exists", func(t *testing.T) {
		if !testItInIRIs(f.Context(), it.Context) {
			t.Errorf("filter %v doesn't contain any of %v", f.Context(), it.Context)
		}
	})
	it.Context = vocab.ItemCollection{vocab.IRI("/objects/test123"), vocab.IRI("https://example.com")}
	t.Run("missing", func(t *testing.T) {
		if testItInIRIs(f.Context(), it.Context) {
			t.Errorf("filter %v shouldn't contain any of %v", f.Context(), it.Context)
		}
	})
}

func TestFilters_InReplyTo(t *testing.T) {
	f := Filters{
		InReplTo: CompStrs{EqualsString("test")},
	}
	if f.InReplyTo() == nil {
		t.Errorf("InReplyTo() should not return nil")
		return
	}
	it := mockItem()
	it.InReplyTo = vocab.ItemCollection{vocab.IRI("/objects/test")}
	t.Run("exists", func(t *testing.T) {
		if !testItInIRIs(f.InReplyTo(), it.InReplyTo) {
			t.Errorf("filter %v doesn't contain any of %v", f.InReplyTo(), it.InReplyTo)
		}
	})
	it.InReplyTo = vocab.ItemCollection{vocab.IRI("/objects/test123"), vocab.IRI("https://example.com")}
	t.Run("missing", func(t *testing.T) {
		if testItInIRIs(f.InReplyTo(), it.InReplyTo) {
			t.Errorf("filter %v shouldn't contain any of %v", f.InReplyTo(), it.InReplyTo)
		}
	})
}

func TestFilters_MediaTypes(t *testing.T) {
	tests := []struct {
		name string
		args Filters
		want []vocab.MimeType
	}{
		{
			name: "empty",
			args: Filters{
				MedTypes: []vocab.MimeType{},
			},
			want: []vocab.MimeType{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.args.MediaTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filters.MediaTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilters_Names(t *testing.T) {
	tests := []struct {
		name string
		args Filters
		want CompStrs
	}{
		{
			name: "empty",
			args: Filters{
				Name: CompStrs{},
			},
			want: CompStrs{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.args.Names(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filters.Names() = %v, want %v", got, tt.want)
			}
		})
	}
}

func mockActivity() vocab.Activity {
	return vocab.Activity{
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
		Actor:        nil,
		Target:       nil,
		Result:       nil,
		Origin:       nil,
		Instrument:   nil,
		Object:       nil,
	}

}
func TestFilters_Objects(t *testing.T) {
	f := Filters{
		Object: &Filters{Key: []Hash{Hash("test")}},
	}
	if f.Objects() == nil {
		t.Errorf("Object() should not return nil")
		return
	}
	act := mockActivity()
	act.Object = vocab.IRI("/objects/test")
	t.Run("exists", func(t *testing.T) {
		if !testItInIRIs(IRIsFilter(f.Objects()...), act.Object) {
			t.Errorf("filter %v doesn't contain any of %v", f.Objects(), act.Object)
		}
	})
	act.Object = vocab.ItemCollection{vocab.IRI("/objects/test123"), vocab.IRI("https://example.com")}
	t.Run("missing", func(t *testing.T) {
		if testItInIRIs(IRIsFilter(f.Objects()...), act.Object) {
			t.Errorf("filter %v shouldn't contain any of %v", f.Objects(), act.Object)
		}
	})
}

func TestFilters_Targets(t *testing.T) {
	f := Filters{
		Target: &Filters{Key: []Hash{Hash("test")}},
	}
	act := mockActivity()
	act.Target = vocab.IRI("/objects/test")
	t.Run("exists", func(t *testing.T) {
		if !testItInIRIs(IRIsFilter(f.Targets()...), act.Target) {
			t.Errorf("filter %v doesn't contain any of %v", f.Targets(), act.Target)
		}
	})
	act.Target = vocab.ItemCollection{vocab.IRI("/objects/example123"), vocab.IRI("https://example.com")}
	t.Run("missing", func(t *testing.T) {
		if testItInIRIs(IRIsFilter(f.Targets()...), act.Target) {
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

func Test_filterNaturalLanguageValues(t *testing.T) {
	type args struct {
		filters CompStrs
		valArr  []vocab.NaturalLanguageValues
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "basic-equality",
			args: args{
				filters: CompStrs{
					CompStr{
						Operator: "",
						Str:      "ana",
					},
				},
				valArr: []vocab.NaturalLanguageValues{
					{
						vocab.LangRefValue{
							Ref:   vocab.NilLangRef,
							Value: vocab.Content("ana"),
						},
					},
				},
			},
			want: true,
		},
		{
			name: "basic-equality-with-nil-first",
			args: args{
				filters: CompStrs{
					CompStr{
						Operator: "",
						Str:      "ana",
					},
				},
				valArr: []vocab.NaturalLanguageValues{
					nil,
					{
						vocab.LangRefValue{
							Ref:   vocab.NilLangRef,
							Value: vocab.Content("ana"),
						},
					},
				},
			},
			want: true,
		},
		{
			name: "basic-like",
			args: args{
				filters: CompStrs{
					CompStr{
						Operator: "~",
						Str:      "ana",
					},
				},
				valArr: []vocab.NaturalLanguageValues{
					{
						vocab.LangRefValue{
							Ref:   vocab.NilLangRef,
							Value: vocab.Content("ana"),
						},
					},
				},
			},
			want: true,
		},
		{
			name: "basic-like-with-longer-value",
			args: args{
				filters: CompStrs{
					CompStr{
						Operator: "~",
						Str:      "ana",
					},
				},
				valArr: []vocab.NaturalLanguageValues{
					{
						vocab.LangRefValue{
							Ref:   vocab.NilLangRef,
							Value: vocab.Content("anathema"),
						},
					},
				},
			},
			want: true,
		},
		{
			name: "basic-different",
			args: args{
				filters: CompStrs{
					CompStr{
						Operator: "!",
						Str:      "ana",
					},
				},
				valArr: []vocab.NaturalLanguageValues{
					{
						vocab.LangRefValue{
							Ref:   vocab.NilLangRef,
							Value: vocab.Content("bob"),
						},
					},
				},
			},
			want: true,
		},
		{
			name: "basic-different-with-empty-values",
			args: args{
				filters: CompStrs{
					CompStr{
						Operator: "!",
						Str:      "ana",
					},
				},
				valArr: []vocab.NaturalLanguageValues{
					nil,
					{},
					{
						vocab.LangRefValue{
							Ref:   vocab.NilLangRef,
							Value: vocab.Content("bob"),
						},
					},
				},
			},
			want: true,
		},
		{
			name: "basic-false-equality",
			args: args{
				filters: CompStrs{
					CompStr{
						Operator: "",
						Str:      "ana",
					},
				},
				valArr: []vocab.NaturalLanguageValues{
					{
						vocab.LangRefValue{
							Ref:   vocab.NilLangRef,
							Value: vocab.Content("bob"),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "basic-false-equality-with-nil-first",
			args: args{
				filters: CompStrs{
					CompStr{
						Operator: "",
						Str:      "ana",
					},
				},
				valArr: []vocab.NaturalLanguageValues{
					nil,
					{
						vocab.LangRefValue{
							Ref:   vocab.NilLangRef,
							Value: vocab.Content("bob"),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "basic-false-like",
			args: args{
				filters: CompStrs{
					CompStr{
						Operator: "~",
						Str:      "ana",
					},
				},
				valArr: []vocab.NaturalLanguageValues{
					{
						vocab.LangRefValue{
							Ref:   vocab.NilLangRef,
							Value: vocab.Content("bob"),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "basic-false-like-with-longer-value",
			args: args{
				filters: CompStrs{
					CompStr{
						Operator: "~",
						Str:      "ana",
					},
				},
				valArr: []vocab.NaturalLanguageValues{
					{
						vocab.LangRefValue{
							Ref:   vocab.NilLangRef,
							Value: vocab.Content("bobsyouruncle"),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "basic-false-different",
			args: args{
				filters: CompStrs{
					CompStr{
						Operator: "!",
						Str:      "ana",
					},
				},
				valArr: []vocab.NaturalLanguageValues{
					{
						vocab.LangRefValue{
							Ref:   vocab.NilLangRef,
							Value: vocab.Content("ana"),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "basic-false-different-with-empty-values",
			args: args{
				filters: CompStrs{
					CompStr{
						Operator: "!",
						Str:      "ana",
					},
				},
				valArr: []vocab.NaturalLanguageValues{
					nil,
					{},
				},
			},
			want: false,
		},
		{
			name: "one value: exact match success",
			args: args{
				CompStrs{StringEquals("ana")},
				[]vocab.NaturalLanguageValues{
					{
						vocab.LangRefValue{
							Ref:   vocab.NilLangRef,
							Value: []byte("ana"),
						},
					},
				},
			},
			want: true,
		},
		{
			name: "one value: exact match failure",
			args: args{
				CompStrs{StringEquals("ana")},
				[]vocab.NaturalLanguageValues{
					{
						vocab.LangRefValue{
							Ref:   vocab.NilLangRef,
							Value: []byte("na"),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "one value: partial match success",
			args: args{
				CompStrs{StringLike("ana")},
				[]vocab.NaturalLanguageValues{
					{
						vocab.LangRefValue{
							Ref:   vocab.NilLangRef,
							Value: []byte("analema"),
						},
					},
				},
			},
			want: true,
		},
		{
			name: "one value: exact match failure",
			args: args{
				CompStrs{StringLike("ana")},
				[]vocab.NaturalLanguageValues{
					{
						vocab.LangRefValue{
							Ref:   vocab.NilLangRef,
							Value: []byte("na"),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "one value: negated match success",
			args: args{
				CompStrs{StringDifferent("ana")},
				[]vocab.NaturalLanguageValues{
					{
						vocab.LangRefValue{
							Ref:   vocab.NilLangRef,
							Value: []byte("lema"),
						},
					},
				},
			},
			want: true,
		},
		{
			name: "one value: negated match failure",
			args: args{
				CompStrs{StringDifferent("ana")},
				[]vocab.NaturalLanguageValues{
					{
						vocab.LangRefValue{
							Ref:   vocab.NilLangRef,
							Value: []byte("ana"),
						},
					},
				},
			},
			want: false,
		},
		// multiple filters
		{
			name: "multi filters: exact match success",
			args: args{
				CompStrs{StringEquals("ana")},
				[]vocab.NaturalLanguageValues{
					{
						vocab.LangRefValue{
							Ref:   vocab.NilLangRef,
							Value: []byte("not-matching"),
						},
						vocab.LangRefValue{
							Ref:   vocab.NilLangRef,
							Value: []byte("ana"),
						},
					},
				},
			},
			want: true,
		},
		{
			name: "multi filters: exact match failure",
			args: args{
				CompStrs{StringEquals("ana")},
				[]vocab.NaturalLanguageValues{
					{
						vocab.LangRefValue{
							Ref:   vocab.NilLangRef,
							Value: []byte("not-matching"),
						},
						vocab.LangRefValue{
							Ref:   vocab.NilLangRef,
							Value: []byte("na"),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "multi filters: partial match success",
			args: args{
				CompStrs{StringLike("ana")},
				[]vocab.NaturalLanguageValues{
					{
						vocab.LangRefValue{
							Ref:   vocab.NilLangRef,
							Value: []byte("not-matching"),
						},
						vocab.LangRefValue{
							Ref:   vocab.NilLangRef,
							Value: []byte("analema"),
						},
					},
				},
			},
			want: true,
		},
		{
			name: "multi filters: exact match failure",
			args: args{
				CompStrs{StringLike("ana")},
				[]vocab.NaturalLanguageValues{
					{
						vocab.LangRefValue{
							Ref:   vocab.NilLangRef,
							Value: []byte("not-matching"),
						},
						vocab.LangRefValue{
							Ref:   vocab.NilLangRef,
							Value: []byte("na"),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "multi filters: negated match success",
			args: args{
				CompStrs{StringDifferent("ana")},
				[]vocab.NaturalLanguageValues{
					{
						vocab.LangRefValue{
							Ref:   vocab.NilLangRef,
							Value: []byte("not-matching"),
						},
						vocab.LangRefValue{
							Ref:   vocab.NilLangRef,
							Value: []byte("lema"),
						},
					},
				},
			},
			want: true,
		},
		{
			name: "multi filters: negated match failure",
			args: args{
				CompStrs{StringDifferent("ana")},
				[]vocab.NaturalLanguageValues{
					{
						vocab.LangRefValue{
							Ref:   vocab.NilLangRef,
							Value: []byte("not-matching"),
						},
						vocab.LangRefValue{
							Ref:   vocab.NilLangRef,
							Value: []byte("ana"),
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filterNaturalLanguageValues(tt.args.filters, tt.args.valArr...); got != tt.want {
				t.Errorf("filterNaturalLanguageValues() = %v, want %v", got, tt.want)
			}
		})
	}
}
