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
		if l.Equal(vocab.IRI(loc)) {
			t.Errorf("Location header value %s, does not match expected value: %s", loc, l)
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
	t := append(res, res.HasContentType(client.ContentTypeJsonLD)...)
	return append(t, func(t testing.TB, r *http.Response) {
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
	})
}

func IsType(typ vocab.ActivityVocabularyType) itemCheckFn {
	return func(t testing.TB, it vocab.Item) {
		if ityp := it.GetType(); !typ.Match(ityp) {
			t.Errorf("Type check failure on item %s, received %s, expected %s", it.GetID(), ityp, typ)
		}
	}
}

func HasContent(cont vocab.NaturalLanguageValues) itemCheckFn {
	return func(t testing.TB, it vocab.Item) {
		err := vocab.OnObject(it, func(ob *vocab.Object) error {
			if !cmp.Equal(cont, ob.Content) {
				t.Errorf("Failed Content check for %s, received %s", ob.ID, cmp.Diff(cont, ob.Content))
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
