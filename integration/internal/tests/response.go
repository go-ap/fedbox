package tests

import (
	"io"
	"net/http"
	"testing"
	"time"

	ct "github.com/elnormous/contenttype"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/client"
	"github.com/go-ap/errors"
	"github.com/go-ap/filters"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type resCheckFn func(*testing.T, *http.Response)

// resChecks represents the expected result of a http request to a FedBOX service
type resChecks []resCheckFn

func (res resChecks) HasCode(c int) resChecks {
	return append(res, func(t *testing.T, response *http.Response) {
		t.Run("Status-Code", func(t *testing.T) {
			if response.StatusCode != c {
				t.Errorf("Invalid status code received %d[%s], expected %d[%s]", response.StatusCode, response.Status, c, http.StatusText(c))
			}
		})
	})
}

func checkItem(wanted vocab.Item, equateFn cmp.Option) bodyCheckFn {
	return func(t *testing.T, raw []byte) {
		got, err := vocab.UnmarshalJSON(raw)
		if err != nil {
			t.Fatalf("Unable to unmarshal ActivityPub object: %v", err)
		}
		if !cmp.Equal(wanted, got, equateFn) {
			t.Errorf("Received item is different %s", cmp.Diff(wanted, got, equateFn))
		}
	}
}

func (res resChecks) HasExactItem(wanted vocab.Item) resChecks {
	return res.BodyMust(checkItem(wanted, equateItems))
}

func areItems(a, b any) bool {
	_, ok1 := a.(vocab.Item)
	_, ok2 := b.(vocab.Item)
	return ok1 && ok2
}

func compareItems(wanted, got any) bool {
	var wi vocab.Item
	var gi vocab.Item
	if w, ok := wanted.(vocab.Item); ok {
		wi = w
	}
	if g, ok := got.(vocab.Item); ok {
		gi = g
	}
	return vocab.ItemsEqual(wi, gi)
}

var equateItems = cmp.FilterValues(areItems, cmp.Comparer(compareItems))

func (res resChecks) HasErrors(wanted ...error) resChecks {
	return res.BodyMust(func(t *testing.T, raw []byte) {
		t.Run("Error body", func(t *testing.T) {
			maybeErr, err := errors.UnmarshalJSON(raw)
			if err != nil {
				t.Errorf("Unable to unmarshal FedBOX error: %v", err)
			}
			if !cmp.Equal(wanted, maybeErr, EquateWeakErrors) {
				t.Errorf("Received error from FedBOX server: %s", cmp.Diff(wanted, maybeErr, EquateWeakErrors))
			}
		})
	})
}

func (res resChecks) HasContentType(validContentTypes ...string) resChecks {
	valid := make([]ct.MediaType, 0, len(validContentTypes))
	for _, c := range validContentTypes {
		valid = append(valid, ct.NewMediaType(c))
	}
	return append(res, func(t *testing.T, r *http.Response) {
		t.Run("Header Content-Type", func(t *testing.T) {
			contentType := r.Header.Get("Content-Type")
			if !ct.NewMediaType(contentType).MatchesAny(valid...) {
				t.Errorf("Wrong Content-Type header '%s', expected one of %#v", contentType, validContentTypes)
			}
		})
	})
}

func (res resChecks) HasLocation(l vocab.IRI) resChecks {
	return append(res, func(t *testing.T, response *http.Response) {
		t.Run("Header Location", func(t *testing.T) {
			loc := response.Header.Get("Location")
			if l != "" && loc == "" {
				t.Errorf("Empty Location header value, does not match expected: %s", l)
				return
			}
			if !l.Equal(vocab.IRI(loc)) {
				t.Errorf("Location header value %s, does not match expected: %s", loc, l)
			}
		})
	})
}

func Response() resChecks {
	return make(resChecks, 0, 4)
}

func (res resChecks) Run(t *testing.T, r *http.Response) {
	for _, check := range res {
		check(t, r)
	}
}

type bodyCheckFn func(*testing.T, []byte)

func (res resChecks) BodyMust(bodyChecks ...bodyCheckFn) resChecks {
	return append(res, func(t *testing.T, r *http.Response) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Unable to read response body: %v", err)
		}
		defer func() {
			if err := r.Body.Close(); err != nil {
				t.Errorf("Unable to close response body: %v", err)
			}
		}()
		for _, checkFn := range bodyChecks {
			checkFn(t, raw)
		}
	})
}

type itemCheckFn func(*testing.T, vocab.Item)

func (res resChecks) ItemMatch(itemChecks ...itemCheckFn) resChecks {
	return append(res.HasContentType(client.ContentTypeJsonLD),
		func(t *testing.T, r *http.Response) {
			t.Run("Item", func(t *testing.T) {
				raw, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("Unable to read response body: %v", err)
				}
				defer func() {
					if err := r.Body.Close(); err != nil {
						t.Errorf("Unable to close response body: %v", err)
					}
				}()

				it, err := vocab.UnmarshalJSON(raw)
				if err != nil {
					t.Errorf("Failed to unmarshal ActivityPub object from body: %v", err)
				}
				if vocab.IsNil(it) && len(itemChecks) > 0 {
					if maybeErr, err1 := errors.UnmarshalJSON(raw); err1 == nil {
						t.Errorf("Received error from FedBOX server: %v", maybeErr)
					} else {
						t.Errorf("Invalid nil item in response when expecting to run checks")
					}
					return
				}
				t.Run(string(it.GetLink()), func(t *testing.T) {
					for _, checkFn := range itemChecks {
						checkFn(t, it)
					}
				})
			})
		},
	)
}

func HasID(want vocab.IRI) itemCheckFn {
	return func(t *testing.T, it vocab.Item) {
		t.Run("ID", func(t *testing.T) {
			if got := it.GetID(); !want.Equal(got) {
				t.Errorf("Received %s, expected %s", got, want)
			}
		})
	}
}

func HasInbox(want vocab.IRI) itemCheckFn {
	return func(t *testing.T, it vocab.Item) {
		t.Run("Inbox", func(t *testing.T) {
			if got := vocab.Inbox.IRI(it); !got.Equal(want) {
				t.Errorf("Received %s, expected %s", got, want)
			}
		})
	}
}

func HasOutbox(want vocab.IRI) itemCheckFn {
	return func(t *testing.T, it vocab.Item) {
		t.Run("Outbox", func(t *testing.T) {
			if got := vocab.Outbox.IRI(it); !got.Equal(want) {
				t.Errorf("Received %s, expected %s", got, want)
			}
		})
	}
}

func IsType(typ vocab.Typer) itemCheckFn {
	return func(t *testing.T, it vocab.Item) {
		t.Run("Type", func(t *testing.T) {
			if ityp := it.GetType(); ityp != typ && !typ.AsTypes().Match(ityp) {
				t.Errorf("Received %s, expected %s", ityp, typ)
			}
		})
	}
}

func nlv[T ~string | vocab.NaturalLanguageValues](c T) vocab.NaturalLanguageValues {
	var result vocab.NaturalLanguageValues
	switch v := any(c).(type) {
	case string:
		result = vocab.DefaultNaturalLanguage(v)
	case []byte:
		result = vocab.DefaultNaturalLanguage(string(v))
	case vocab.NaturalLanguageValues:
		result = v
	}
	return result
}

func HasPreferredUsername[T ~string | vocab.NaturalLanguageValues](name T) itemCheckFn {
	v := nlv(name)
	return func(t *testing.T, it vocab.Item) {
		t.Run("PreferredUsername", func(t *testing.T) {
			err := vocab.OnActor(it, func(act *vocab.Actor) error {
				if !cmp.Equal(v, act.PreferredUsername) {
					t.Errorf("Diff %s", cmp.Diff(v, act.PreferredUsername))
				}
				return nil
			})
			if err != nil {
				t.Errorf("Failed Actor %s check: %v", it.GetID(), err)
			}
		})
	}
}

func HasSummary[T ~string | vocab.NaturalLanguageValues](cont T) itemCheckFn {
	v := nlv(cont)
	return func(t *testing.T, it vocab.Item) {
		t.Run("Summary", func(t *testing.T) {
			err := vocab.OnObject(it, func(ob *vocab.Object) error {
				if !cmp.Equal(v, ob.Summary) {
					t.Errorf("Diff %s", cmp.Diff(v, ob.Summary))
				}
				return nil
			})
			if err != nil {
				t.Errorf("Failed Object %s check: %v", it.GetID(), err)
			}
		})
	}
}

func HasName[T ~string | vocab.NaturalLanguageValues](cont T) itemCheckFn {
	v := nlv(cont)
	return func(t *testing.T, it vocab.Item) {
		t.Run("Name", func(t *testing.T) {
			err := vocab.OnObject(it, func(ob *vocab.Object) error {
				if !cmp.Equal(v, ob.Name) {
					t.Errorf("Diff %s", cmp.Diff(v, ob.Name))
				}
				return nil
			})
			if err != nil {
				t.Errorf("Failed Object %s check: %v", it.GetID(), err)
			}
		})
	}
}

func HasContent[T ~string | vocab.NaturalLanguageValues](cont T) itemCheckFn {
	v := nlv(cont)
	return func(t *testing.T, it vocab.Item) {
		t.Run("Content", func(t *testing.T) {
			err := vocab.OnObject(it, func(ob *vocab.Object) error {
				if !cmp.Equal(v, ob.Content) {
					t.Errorf("Diff %s", cmp.Diff(v, ob.Content))
				}
				return nil
			})
			if err != nil {
				t.Errorf("Failed Object %s check: %v", it.GetID(), err)
			}
		})
	}
}

func HasMediaType(mt vocab.MimeType) itemCheckFn {
	return func(t *testing.T, it vocab.Item) {
		t.Run("MediaType", func(t *testing.T) {
			err := vocab.OnObject(it, func(ob *vocab.Object) error {
				if !cmp.Equal(mt, ob.MediaType) {
					t.Errorf("Diff %s", cmp.Diff(mt, ob.MediaType))
				}
				return nil
			})
			if err != nil {
				t.Errorf("Failed Object %s check: %v", it.GetID(), err)
			}
		})
	}
}
func HasSource[T ~string | vocab.NaturalLanguageValues](cont T, mt vocab.MimeType) itemCheckFn {
	v := nlv(cont)
	return func(t *testing.T, it vocab.Item) {
		t.Run("Source", func(t *testing.T) {
			err := vocab.OnObject(it, func(ob *vocab.Object) error {
				if !cmp.Equal(v, ob.Source.Content) {
					t.Errorf("Content Diff %s", cmp.Diff(v, ob.Source.Content))
				}
				if !cmp.Equal(mt, ob.Source.MediaType) {
					t.Errorf("MediaType Diff %s", cmp.Diff(mt, ob.Source.MediaType))
				}
				return nil
			})
			if err != nil {
				t.Errorf("Failed Object %s check: %v", it.GetID(), err)
			}
		})
	}
}

func HasTo(to vocab.Item) itemCheckFn {
	return func(t *testing.T, it vocab.Item) {
		t.Run("To", func(t *testing.T) {
			err := vocab.OnObject(it, func(ob *vocab.Object) error {
				if !cmp.Equal(to, ob.To, equateItems) {
					t.Errorf("Received %s, expected %s", ob.To, to)
				}
				return nil
			})
			if err != nil {
				t.Errorf("Invalid Object: %v", err)
			}
		})
	}
}

func HasCC(cc vocab.Item) itemCheckFn {
	return func(t *testing.T, it vocab.Item) {
		t.Run("CC", func(t *testing.T) {
			err := vocab.OnObject(it, func(ob *vocab.Object) error {
				if !cmp.Equal(cc, ob.CC, equateItems) {
					t.Errorf("Received %s, expected %s", ob.CC, cc)
				}
				return nil
			})
			if err != nil {
				t.Errorf("Invalid Object: %v", err)
			}
		})
	}
}

func WasUpdated(d time.Time) itemCheckFn {
	return func(t *testing.T, it vocab.Item) {
		t.Run("Updated", func(t *testing.T) {
			err := vocab.OnObject(it, func(ob *vocab.Object) error {
				if !cmp.Equal(d, ob.Updated, cmpopts.EquateApproxTime(time.Second)) {
					t.Errorf("Failed Updated date check for %s, received %s, expected %s", ob.ID, ob.Updated, d)
				}
				return nil
			})
			if err != nil {
				t.Errorf("Invalid Object: %v", err)
			}
		})
	}
}

func WasPublished(d time.Time) itemCheckFn {
	return func(t *testing.T, it vocab.Item) {
		t.Run("Published", func(t *testing.T) {
			err := vocab.OnObject(it, func(ob *vocab.Object) error {
				if !cmp.Equal(d, ob.Published, cmpopts.EquateApproxTime(2*time.Second)) {
					t.Errorf("Received %s, expected %s", ob.Published, d)
				}
				return nil
			})
			if err != nil {
				t.Errorf("Invalid Object: %v", err)
			}
		})
	}
}

func HasAudience(u vocab.Item) itemCheckFn {
	return func(t *testing.T, it vocab.Item) {
		t.Run("Audience", func(t *testing.T) {
			err := vocab.OnObject(it, func(ob *vocab.Object) error {
				if !cmp.Equal(ob.Audience, u, equateItems) {
					t.Errorf("%s", cmp.Diff(u, ob.Audience, equateItems))
				}
				return nil
			})
			if err != nil {
				t.Errorf("Invalid Object: %v", err)
			}
		})
	}
}

func HasAttributedTo(u vocab.Item) itemCheckFn {
	return func(t *testing.T, it vocab.Item) {
		t.Run("AttributedTo", func(t *testing.T) {
			err := vocab.OnObject(it, func(ob *vocab.Object) error {
				if !cmp.Equal(ob.AttributedTo, u, equateItems) {
					t.Errorf("%s", cmp.Diff(u, ob.AttributedTo, equateItems))
				}
				return nil
			})
			if err != nil {
				t.Errorf("Invalid Object: %v", err)
			}
		})
	}
}

func HasURL(u vocab.Item) itemCheckFn {
	return func(t *testing.T, it vocab.Item) {
		t.Run("URL", func(t *testing.T) {
			err := vocab.OnObject(it, func(ob *vocab.Object) error {
				if !cmp.Equal(ob.URL, u, equateItems) {
					t.Errorf("%s", cmp.Diff(u, ob.URL, equateItems))
				}
				return nil
			})
			if err != nil {
				t.Errorf("Invalid Object: %v", err)
			}
		})
	}
}

func HasTag(u vocab.Item) itemCheckFn {
	return func(t *testing.T, it vocab.Item) {
		t.Run("Tag", func(t *testing.T) {
			err := vocab.OnObject(it, func(ob *vocab.Object) error {
				if !cmp.Equal(ob.Tag, u, equateItems) {
					t.Errorf("%s", cmp.Diff(u, ob.Tag, equateItems))
				}
				return nil
			})
			if err != nil {
				t.Errorf("Invalid Object: %v", err)
			}
		})
	}
}

func HasActor(iri vocab.IRI) itemCheckFn {
	return func(t *testing.T, it vocab.Item) {
		t.Run("Actor", func(t *testing.T) {
			err := vocab.OnIntransitiveActivity(it, func(act *vocab.IntransitiveActivity) error {
				if !iri.Equal(act.Actor.GetLink()) {
					t.Errorf("Received %s, expected %s", act.Actor.GetID(), iri)
				}
				return nil
			})
			if err != nil {
				t.Errorf("Invalid IntransitiveActivity: %v", err)
			}
		})
	}
}

func HasProxyURL(iri vocab.IRI) itemCheckFn {
	return func(t *testing.T, it vocab.Item) {
		t.Run("ProxyURL", func(t *testing.T) {
			err := vocab.OnActor(it, func(act *vocab.Actor) error {
				if act.Endpoints == nil {
					t.Errorf("Actor's Endpoints is nil")
					return nil
				}
				if !cmp.Equal(iri, act.Endpoints.ProxyURL, equateItems) {
					t.Errorf("Received %s, expected %s", act.Endpoints.ProxyURL, iri)
				}
				return nil
			})
			if err != nil {
				t.Errorf("Invalid Actor: %v", err)
			}
		})
	}
}

func HasSharedInbox(iri vocab.Item) itemCheckFn {
	return func(t *testing.T, it vocab.Item) {
		t.Run("SharedInbox", func(t *testing.T) {
			err := vocab.OnActor(it, func(act *vocab.Actor) error {
				if act.Endpoints == nil {
					t.Errorf("Actor's Endpoints is nil")
					return nil
				}
				if !cmp.Equal(iri, act.Endpoints.SharedInbox, equateItems) {
					t.Errorf("Received %s, expected %s", act.Endpoints.SharedInbox, iri)
				}
				return nil
			})
			if err != nil {
				t.Errorf("Invalid Actor: %v", err)
			}
		})
	}
}

func HasObject(iri vocab.IRI) itemCheckFn {
	return func(t *testing.T, it vocab.Item) {
		t.Run("Object", func(t *testing.T) {
			err := vocab.OnActivity(it, func(act *vocab.Activity) error {
				if !iri.Equal(act.Object.GetLink()) {
					t.Errorf("Received %s, expected %s", act.Object.GetID(), iri)
				}
				return nil
			})
			if err != nil {
				t.Errorf("Invalid Activity: %v", err)
			}
		})
	}
}

func HasTotalItems(cnt int) itemCheckFn {
	return func(t *testing.T, it vocab.Item) {
		t.Run("TotalItems", func(t *testing.T) {
			err := vocab.OnOrderedCollection(it, func(col *vocab.OrderedCollection) error {
				if int(col.TotalItems) != cnt {
					t.Errorf("Received %d, expected %d", col.TotalItems, cnt)
				}
				return nil
			})
			if err != nil {
				t.Errorf("Invalid Collection: %v", err)
			}
		})
	}
}

func HasItem(it vocab.Item) itemCheckFn {
	return func(t *testing.T, got vocab.Item) {
		t.Run("Contains:"+string(it.GetLink()), func(t *testing.T) {
			err := vocab.OnOrderedCollection(got, func(col *vocab.OrderedCollection) error {
				gotItems := col.OrderedItems
				maybeFound, _ := filters.Checks{filters.SameID(it.GetID())}.Run(gotItems).(vocab.ItemCollection)
				if maybeFound.Count() == 0 {
					t.Errorf("Item does not exist in collection: %s", it.GetLink())
					return nil
				}
				gotIt := maybeFound.First()
				if !it.GetLink().Equal(gotIt.GetLink()) {
					t.Errorf("Item differs: %s", cmp.Diff(it.GetLink(), gotIt.GetLink()))
					return nil
				}
				return nil
			})
			if err != nil {
				t.Errorf("Invalid Collection: %v", err)
			}
		})
	}
}

func HasExactItems(items ...vocab.Item) itemCheckFn {
	return func(t *testing.T, got vocab.Item) {
		t.Run("Items", func(t *testing.T) {
			err := vocab.OnOrderedCollection(got, func(col *vocab.OrderedCollection) error {
				gotItems := col.OrderedItems
				if len(gotItems) != len(items) {
					t.Errorf("The amount of received items %d does not match the expected count: %d", len(gotItems), len(items))
					return nil
				}
				for i, it := range items {
					maybeFound, _ := filters.Checks{filters.SameID(it.GetID())}.Run(gotItems).(vocab.ItemCollection)
					if maybeFound.Count() == 0 {
						t.Errorf("Item at pos %d does not exist in collection: %s", i, it.GetLink())
						return nil
					}
					gotIt := maybeFound.First()
					if !it.GetLink().Equal(gotIt.GetLink()) {
						t.Errorf("Item at pos %d differs: %s", i, cmp.Diff(it.GetLink(), gotIt.GetLink()))
						return nil
					}
				}
				return nil
			})
			if err != nil {
				t.Errorf("Invalid Collection: %v", err)
			}
		})
	}
}
