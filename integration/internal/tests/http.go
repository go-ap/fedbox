package tests

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"testing"
	"time"

	ct "github.com/elnormous/contenttype"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	c "github.com/go-ap/fedbox/integration/internal/containers"
	"github.com/google/go-cmp/cmp"
)

type resCheckFn func(testing.TB, *http.Response)

// resChecks represents the expected result of a http request to a FedBOX service
type resChecks []resCheckFn

type resChecksInitFn func(*resChecks)

func HasCode(c int) resChecksInitFn {
	return func(res *resChecks) {
		*res = append(*res, func(t testing.TB, response *http.Response) {
			if response.StatusCode == c {
				return
			}
			t.Errorf("Invalid status code received %d[%s], expected %d[%s]", response.StatusCode, response.Status, c, http.StatusText(c))
		})
	}
}

func checkItem(wanted vocab.Item, equateFn cmp.Option) resCheckFn {
	return func(t testing.TB, r *http.Response) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Unable to read HTTP body (read %d): %v", len(raw), err)
		}
		defer r.Body.Close()
		got, err := vocab.UnmarshalJSON(raw)
		if err != nil {
			t.Fatalf("Unable to unmarshal ActivityPub object: %v", err)
		}
		if !cmp.Equal(wanted, got, equateFn) {
			t.Errorf("Received item is different %s", cmp.Diff(wanted, got, equateFn))
		}
	}
}

func HasItemProperties(wanted vocab.Item) resChecksInitFn {
	return func(res *resChecks) {
		*res = append(*res, checkItem(wanted, equateItemsWithoutID))
	}
}

func HasExactItem(wanted vocab.Item) resChecksInitFn {
	return func(res *resChecks) {
		*res = append(*res, checkItem(wanted, equateItems))
	}
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

func compareItemsWithoutID(wanted, got any) bool {
	var wi vocab.Item
	var gi vocab.Item
	if w, ok := wanted.(vocab.Item); ok {
		wi = w
	}
	if g, ok := got.(vocab.Item); ok {
		gi = g
	}
	vocab.OnObject(gi, func(ob *vocab.Object) error {
		if ob.ID == "" {
			ob.ID = wi.GetID()
		}
		return nil
	})
	return vocab.ItemsEqual(wi, gi)
}

var equateItemsWithoutID = cmp.FilterValues(areItems, cmp.Comparer(compareItemsWithoutID))

func HasErrors(wanted ...error) resChecksInitFn {
	return func(res *resChecks) {
		*res = append(*res, func(t testing.TB, r *http.Response) {
			t.Helper()
			raw, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("Unable to read HTTP body (read %d): %v", len(raw), err)
			}
			defer r.Body.Close()

			maybeErr, err := errors.UnmarshalJSON(raw)
			if err != nil {
				t.Fatalf("Unable to unmarshal FedBOX error: %v", err)
			}
			if !cmp.Equal(wanted, maybeErr, equateWeakErrors) {
				t.Errorf("Received error from FedBOX server: %s", cmp.Diff(wanted, maybeErr, equateWeakErrors))
			}
		})
	}
}

func HasContentType(validContentTypes ...string) resChecksInitFn {
	valid := make([]ct.MediaType, 0, len(validContentTypes))
	for _, c := range validContentTypes {
		valid = append(valid, ct.NewMediaType(c))
	}
	return func(res *resChecks) {
		*res = append(*res, func(t testing.TB, r *http.Response) {
			contentType := r.Header.Get("Content-Type")
			if !ct.NewMediaType(contentType).MatchesAny(valid...) {
				t.Errorf("Wrong Content-Type header '%s', expected one of %#v", contentType, validContentTypes)
			}
		})
	}
}

func HasLocation(l vocab.IRI) resChecksInitFn {
	return func(res *resChecks) {
		*res = append(*res, func(t testing.TB, response *http.Response) {
			loc := response.Header.Get("Location")
			if l.Equal(vocab.IRI(loc)) {
				t.Errorf("Location header value %s, does not match expected value: %s", loc, l)
			}
		})
	}
}

func Response(initFn ...resChecksInitFn) resChecks {
	s := make(resChecks, 0, len(initFn))
	for _, fn := range initFn {
		fn(&s)
	}
	return s
}

func (res resChecks) Run(t *testing.T, r *http.Response) {
	t.Helper()
	for _, check := range res {
		check(t, r)
	}
}

type HTTPTest struct {
	Name string
	Req  *builder
	Res  resChecks
}

func (ht HTTPTest) Label() string {
	return ht.Name
}

var httpClient = http.Client{
	Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
}

func (ht HTTPTest) Fn(ctx context.Context, mocks c.Running) func(t *testing.T) {
	return func(t *testing.T) {
		ht.Run(ctx, mocks, t)
	}
}

func (ht HTTPTest) Run(ctx context.Context, mocks c.Running, t *testing.T) {
	cl := &httpClient

	var cancelFn func()
	ctx, cancelFn = context.WithTimeout(ctx, 10*time.Second)
	defer cancelFn()

	req, err := ht.Req.Request(ctx, mocks)
	if err != nil {
		t.Fatalf("unable to create request: %+v", err)
	}

	resp, err := cl.Do(req)
	if err != nil {
		t.Errorf("\t[%s]%s", req.Method, req.URL.String())
		t.Errorf("\t%+v", req.Header)
		t.Fatalf("failed request %+v", err)
	}

	ht.Res.Run(t, resp)
}

var _ RunnableTest = HTTPTest{}
