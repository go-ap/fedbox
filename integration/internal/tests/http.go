package tests

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"slices"
	"testing"
	"time"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/client"
	"github.com/go-ap/errors"
	c "github.com/go-ap/fedbox/integration/internal/containers"
	"github.com/google/go-cmp/cmp"
)

type testRequest []reqInitFn

func (req testRequest) build(ctx context.Context, mocks c.Running, t testing.TB) (*http.Request, error) {
	r := new(http.Request)
	r.Header = make(http.Header)
	for _, fn := range req {
		fn(r, t)
	}
	err := mocks.BuildRequest(ctx, r)
	return r, err
}

type reqInitFn func(*http.Request, testing.TB)

func WithURL[T string | vocab.IRI](u T) reqInitFn {
	return func(r *http.Request, t testing.TB) {
		r.URL, _ = url.Parse(string(u))
		r.Header.Add("Host", r.URL.Hostname())
	}
}

func WithMethod(m string) reqInitFn {
	return func(r *http.Request, t testing.TB) {
		r.Method = m
	}
}

func WithBody(b io.Reader) reqInitFn {
	return func(r *http.Request, t testing.TB) {
		r.Body = io.NopCloser(b)
	}
}

func WithHeader(k, v string) reqInitFn {
	return func(r *http.Request, t testing.TB) {
		r.Header.Add(k, v)
	}
}

var MockDate = time.Date(2001, time.April, 1, 0, 0, 23, 00, time.UTC)

func WithSigner(signFn func(*http.Request) error) reqInitFn {
	return func(r *http.Request, t testing.TB) {
		if hasDate := r.Header.Get("Date"); hasDate == "" {
			r.Header.Add("Date", MockDate.Format(http.TimeFormat))
		}
		if err := signFn(r); err != nil {
			t.Errorf("Unable to sign request: %v", err)
		}
	}
}

func noS() string { return "" }

type nilReader struct{}

var _ io.Reader = nilReader{}

func (n nilReader) Read(p []byte) (int, error) {
	return len(p), nil
}

func Request(initFn ...reqInitFn) testRequest {
	return initFn
}

// testResponse represents the expected result of a http request to a FedBOX service
type testResponse struct {
	code int
	it   vocab.Item
	err  []error
}

type resInitFn func(*testResponse)

func HasCode(c int) resInitFn {
	return func(res *testResponse) {
		res.code = c
	}
}

func HasItem(it vocab.Item) resInitFn {
	return func(res *testResponse) {
		res.it = it
	}
}

func HasErrors(err ...error) resInitFn {
	return func(res *testResponse) {
		res.err = err
	}
}

func Response(initFn ...resInitFn) testResponse {
	s := testResponse{code: 0x000}
	for _, fn := range initFn {
		fn(&s)
	}
	return s
}

func (res testResponse) validate(t *testing.T, r *http.Response) {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("Unable to read HTTP body (read %d): %v", len(raw), err)
	}
	defer r.Body.Close()

	if res.code != r.StatusCode {
		t.Errorf("Invalid status code received %d, expected %d", r.StatusCode, res.code)
	}

	if res.it != nil {
		contentType := r.Header.Get("Content-Type")
		validContentTypes := []string{client.ContentTypeJsonLD, client.ContentTypeJsonActivity, client.ContentTypeJson}
		if !slices.Contains(validContentTypes, contentType) {
			t.Errorf("Wrong Content-Type header '%s', expected %+v", contentType, validContentTypes)
		}
		it, err := vocab.UnmarshalJSON(raw)
		if err != nil {
			t.Fatalf("Unable to unmarshal ActivityPub object: %v", err)
		}
		if !cmp.Equal(res.it, it) {
			t.Errorf("Received item is different %s", cmp.Diff(res.it, it))
		}
		return
	}
	maybeErr, err := errors.UnmarshalJSON(raw)
	if err != nil {
		t.Fatalf("Unable to unmarshal FedBOX error: %v", err)
	}
	if !cmp.Equal(res.err, maybeErr, equateWeakErrors) {
		t.Errorf("Received error from FedBOX server: %s", cmp.Diff(res.err, maybeErr, equateWeakErrors))
	}
}

type HTTPTest struct {
	Name string
	Req  testRequest
	Res  testResponse
}

func (pair HTTPTest) Label() string {
	return pair.Name
}

var httpClient = http.Client{
	Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
}

func (pair HTTPTest) Fn(ctx context.Context, mocks c.Running) func(t *testing.T) {
	return func(t *testing.T) {
		pair.Run(ctx, mocks, t)
	}
}

func (pair HTTPTest) Run(ctx context.Context, mocks c.Running, t *testing.T) {
	cl := &httpClient

	var cancelFn func()
	ctx, cancelFn = context.WithTimeout(ctx, 10*time.Second)
	defer cancelFn()

	req, err := pair.Req.build(ctx, mocks, t)
	if err != nil {
		t.Fatalf("unable to create request: %+v", err)
	}

	resp, err := cl.Do(req)
	if err != nil {
		t.Fatalf("Err received on %s->%s: %+v", req.Method, req.URL.String(), err)
	}

	pair.Res.validate(t, resp)
}

var _ RunnableTest = HTTPTest{}
