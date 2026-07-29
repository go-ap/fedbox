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
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type resCheckFn func(testing.TB, *http.Response)

// resChecks represents the expected result of a http request to a FedBOX service
type resChecks []resCheckFn

func (res resChecks) HasCode(c int) resChecks {
	return append(res, func(t testing.TB, response *http.Response) {
		if response.StatusCode == c {
			return
		}
		t.Errorf("Invalid status code received %d[%s], expected %d[%s]", response.StatusCode, response.Status, c, http.StatusText(c))
	})
}

func checkItem(wanted vocab.Item, equateFn cmp.Option) bodyCheckFn {
	return func(t testing.TB, raw []byte) {
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
	return res.BodyMust(func(t testing.TB, raw []byte) {
		maybeErr, err := errors.UnmarshalJSON(raw)
		if err != nil {
			t.Errorf("Unable to unmarshal FedBOX error: %v", err)
		}
		if !cmp.Equal(wanted, maybeErr, EquateWeakErrors) {
			t.Errorf("Received error from FedBOX server: %s", cmp.Diff(wanted, maybeErr, EquateWeakErrors))
		}
	})
}

func (res resChecks) HasContentType(validContentTypes ...string) resChecks {
	valid := make([]ct.MediaType, 0, len(validContentTypes))
	for _, c := range validContentTypes {
		valid = append(valid, ct.NewMediaType(c))
	}
	return append(res, func(t testing.TB, r *http.Response) {
		contentType := r.Header.Get("Content-Type")
		if !ct.NewMediaType(contentType).MatchesAny(valid...) {
			t.Errorf("Wrong Content-Type header '%s', expected one of %#v", contentType, validContentTypes)
		}
	})
}

func (res resChecks) HasLocation(l vocab.IRI) resChecks {
	return append(res, func(t testing.TB, response *http.Response) {
		loc := response.Header.Get("Location")
		if l != "" && loc == "" {
			t.Errorf("Empty Location header value, does not match expected: %s", l)
			return
		}
		if !l.Equal(vocab.IRI(loc)) {
			t.Errorf("Location header value %s, does not match expected: %s", loc, l)
		}
	})
}

func Response() resChecks {
	s := make(resChecks, 0, 4)
	return s
}

func (res resChecks) Run(t *testing.T, r *http.Response) {
	t.Helper()
	for _, check := range res {
		check(t, r)
	}
}

type bodyCheckFn func(testing.TB, []byte)

func (res resChecks) BodyMust(bodyChecks ...bodyCheckFn) resChecks {
	return append(res, func(t testing.TB, r *http.Response) {
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

type itemCheckFn func(testing.TB, vocab.Item)

func (res resChecks) ItemMatch(itemChecks ...itemCheckFn) resChecks {
	return append(res.HasContentType(client.ContentTypeJsonLD),
		func(t testing.TB, r *http.Response) {
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
			for _, checkFn := range itemChecks {
				checkFn(t, it)
			}
		},
	)
}

func HasID(want vocab.IRI) itemCheckFn {
	return func(t testing.TB, it vocab.Item) {
		if got := it.GetID(); !want.Equal(got) {
			t.Errorf("Object ID check failure on item, received %s, expected %s", got, want)
		}
	}
}

func IsType(typ vocab.ActivityVocabularyType) itemCheckFn {
	return func(t testing.TB, it vocab.Item) {
		if ityp := it.GetType(); !typ.Match(ityp) {
			t.Errorf("Type check failure on item %s, received %s, expected %s", it.GetID(), ityp, typ)
		}
	}
}

func nlv[T ~string | vocab.NaturalLanguageValues](c T) vocab.NaturalLanguageValues {
	var result vocab.NaturalLanguageValues
	switch v := any(c).(type) {
	case string:
		result = vocab.DefaultNaturalLanguage[string](v)
	case []byte:
		result = vocab.DefaultNaturalLanguage(string(v))
	case vocab.NaturalLanguageValues:
		result = v
	}
	return result
}

func HasPreferredUsername[T ~string | vocab.NaturalLanguageValues](name T) itemCheckFn {
	v := nlv(name)
	return func(t testing.TB, it vocab.Item) {
		err := vocab.OnActor(it, func(act *vocab.Actor) error {
			if !cmp.Equal(v, act.PreferredUsername) {
				t.Errorf("Failed PreferredUsername check for %s, received %s", act.ID, cmp.Diff(v, act.PreferredUsername))
			}
			return nil
		})
		if err != nil {
			t.Errorf("Failed Actor %s check: %v", it.GetID(), err)
		}
	}
}

func HasContent[T ~string | vocab.NaturalLanguageValues](cont T) itemCheckFn {
	v := nlv(cont)
	return func(t testing.TB, it vocab.Item) {
		err := vocab.OnObject(it, func(ob *vocab.Object) error {
			if !cmp.Equal(v, ob.Content) {
				t.Errorf("Failed Content check for %s, received %s", ob.ID, cmp.Diff(v, ob.Content))
			}
			return nil
		})
		if err != nil {
			t.Errorf("Failed Object %s check: %v", it.GetID(), err)
		}
	}
}

func HasTo(to vocab.Item) itemCheckFn {
	return func(t testing.TB, it vocab.Item) {
		err := vocab.OnObject(it, func(ob *vocab.Object) error {
			if !cmp.Equal(to, ob.To, equateItems) {
				t.Errorf("Failed To check for %s, received %s, expected %s", ob.ID, ob.To, to)
			}
			return nil
		})
		if err != nil {
			t.Errorf("Failed Object %s check: %v", it.GetID(), err)
		}
	}
}

func HasCC(cc vocab.Item) itemCheckFn {
	return func(t testing.TB, it vocab.Item) {
		err := vocab.OnObject(it, func(ob *vocab.Object) error {
			if !cmp.Equal(cc, ob.CC, equateItems) {
				t.Errorf("Failed CC check for %s, received %s, expected %s", ob.ID, ob.CC, cc)
			}
			return nil
		})
		if err != nil {
			t.Errorf("Failed Object %s check: %v", it.GetID(), err)
		}
	}
}

func WasUpdated(d time.Time) itemCheckFn {
	return func(t testing.TB, it vocab.Item) {
		err := vocab.OnObject(it, func(ob *vocab.Object) error {
			if !cmp.Equal(d, ob.Updated, cmpopts.EquateApproxTime(time.Second)) {
				t.Errorf("Failed Updated date check for %s, received %s, expected %s", ob.ID, ob.Updated, d)
			}
			return nil
		})
		if err != nil {
			t.Errorf("Failed Object %s check: %v", it.GetID(), err)
		}
	}
}

func WasPublished(d time.Time) itemCheckFn {
	return func(t testing.TB, it vocab.Item) {
		err := vocab.OnObject(it, func(ob *vocab.Object) error {
			if !cmp.Equal(d, ob.Published, cmpopts.EquateApproxTime(time.Second)) {
				t.Errorf("Failed Published date check for %s, received %s, expected %s", ob.ID, ob.Published, d)
			}
			return nil
		})
		if err != nil {
			t.Errorf("Failed Object %s check: %v", it.GetID(), err)
		}
	}
}

func HasURL(u string) itemCheckFn {
	return func(t testing.TB, it vocab.Item) {
		err := vocab.OnObject(it, func(ob *vocab.Object) error {
			return vocab.OnItem(ob.URL, func(it vocab.Item) error {
				if !it.GetLink().Equal(vocab.IRI(u)) {
					t.Errorf("Object URL %v does not match expected: %s", it, u)
				}
				return nil
			})
		})
		if err != nil {
			t.Errorf("Failed Object %s check: %v", it.GetID(), err)
		}
	}
}

func HasActor(iri vocab.IRI) itemCheckFn {
	return func(t testing.TB, it vocab.Item) {
		err := vocab.OnActivity(it, func(act *vocab.Activity) error {
			if !iri.Equal(act.Actor.GetLink()) {
				t.Errorf("Failed Actor check for %s, received %s, expected %s", act.ID, act.Actor.GetID(), iri)
			}
			return nil
		})
		if err != nil {
			t.Errorf("Failed Activity %s check: %v", it.GetID(), err)
		}
	}
}

func HasObject(iri vocab.IRI) itemCheckFn {
	return func(t testing.TB, it vocab.Item) {
		err := vocab.OnActivity(it, func(act *vocab.Activity) error {
			if !iri.Equal(act.Object.GetLink()) {
				t.Errorf("Failed Object check for %s, received %s, expected %s", act.ID, act.Object.GetID(), iri)
			}
			return nil
		})
		if err != nil {
			t.Errorf("Failed Activity %s check: %v", it.GetID(), err)
		}
	}
}
