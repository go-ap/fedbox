package tests

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"testing"
	"time"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/client"
	c "github.com/go-ap/fedbox/integration/internal/containers"
	"github.com/go-ap/jsonld"
	"github.com/google/go-cmp/cmp"
)

type testRequest struct {
	met     string
	headers http.Header
	urlFn   func() string
	body    io.Reader
}

func (req *testRequest) build(ctx context.Context, mocks c.Running) (*http.Request, error) {
	return mocks.BuildRequest(ctx, req.met, req.urlFn(), nil)
}

type reqInitFn func(*testRequest)

func WithURL(u string) reqInitFn {
	return func(t *testRequest) {
		t.urlFn = func() string {
			return u
		}
	}
}

func WithMethod(m string) reqInitFn {
	return func(t *testRequest) {
		t.met = m
	}
}

func WithBody(r io.Reader) reqInitFn {
	w := bytes.Buffer{}
	return func(t *testRequest) {
		_, _ = io.Copy(&w, r)
		t.body = &w
	}
}

func noS() string { return "" }

type nilReader struct{}

var _ io.Reader = nilReader{}

func (n nilReader) Read(p []byte) (int, error) {
	return len(p), nil
}

func Request(initFn ...reqInitFn) testRequest {
	r := testRequest{
		met:     http.MethodGet,
		headers: make(http.Header),
		urlFn:   noS,
		body:    nilReader{},
	}
	for _, fn := range initFn {
		fn(&r)
	}
	return r
}

// testResponse represents the expected result of a http request to a FedBOX service
type testResponse struct {
	code int
	it   vocab.Item
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

func Response(initFn ...resInitFn) testResponse {
	s := testResponse{code: http.StatusNotImplemented}
	for _, fn := range initFn {
		fn(&s)
	}
	return s
}

func (res testResponse) validate(t *testing.T, r *http.Response) {
	if res.code != r.StatusCode {
		t.Fatalf("Invalid status code received %d, expected :%d", r.StatusCode, res.code)
	}

	raw, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("Error: invalid HTTP body! Read %d: %s", len(raw), err)
	}

	contentType := r.Header.Get("Content-Type")
	if contentType == jsonld.ContentType || contentType == client.ContentTypeActivityJson {
		it, err := vocab.UnmarshalJSON(raw)
		if err != nil {
			t.Fatalf("Error: invalid HTTP body! Read %d: %s", len(raw), err)
		}
		if !cmp.Equal(res.it, it) {
			t.Errorf("Received item is different %s", cmp.Diff(res.it, it))
		}
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
	cl := client.New(client.WithHTTPClient(&httpClient), client.SkipTLSValidation(true))
	var cancelFn func()
	ctx, cancelFn = context.WithTimeout(ctx, 2*time.Second)
	defer cancelFn()

	req, err := pair.Req.build(ctx, mocks)
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
