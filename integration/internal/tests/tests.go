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
	"github.com/go-ap/fedbox/integration/internal/containers"
	"github.com/go-ap/jsonld"
	"github.com/google/go-cmp/cmp"
)

type input struct {
	met     string
	headers http.Header
	urlFn   func() string
	body    io.Reader
}

func (req *input) Request(ctx context.Context, mocks containers.Running) (*http.Request, error) {
	return mocks.BuildRequest(ctx, req.met, req.urlFn(), nil)
}

type reqInitFn func(*input)

func WithURL(u string) reqInitFn {
	return func(t *input) {
		t.urlFn = func() string {
			return u
		}
	}
}

func WithMethod(m string) reqInitFn {
	return func(t *input) {
		t.met = m
	}
}

func WithBody(r io.Reader) reqInitFn {
	w := bytes.Buffer{}
	return func(t *input) {
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

func IN(initFn ...reqInitFn) input {
	r := input{
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

// output represents the expected result of a http request to a FedBOX service
type output struct {
	code int
	it   vocab.Item
}

type resInitFn func(*output)

func HasCode(c int) resInitFn {
	return func(res *output) {
		res.code = c
	}
}

func HasItem(it vocab.Item) resInitFn {
	return func(res *output) {
		res.it = it
	}
}

func OUT(initFn ...resInitFn) output {
	s := output{code: http.StatusNotImplemented}
	for _, fn := range initFn {
		fn(&s)
	}
	return s
}

func (res output) assertResponse(t *testing.T, r *http.Response) {
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

type IOTest struct {
	Name string
	IN   input
	OUT  output
}

var httpClient = http.Client{
	Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
}

func (test IOTest) Run(ctx context.Context, mocks containers.Running) func(t *testing.T) {
	cl := client.New(client.WithHTTPClient(&httpClient), client.SkipTLSValidation(true))
	return func(t *testing.T) {
		var cancelFn func()
		ctx, cancelFn = context.WithTimeout(ctx, 2*time.Second)
		defer cancelFn()

		req, err := test.IN.Request(ctx, mocks)
		if err != nil {
			t.Fatalf("unable to create request: %+v", err)
		}

		resp, err := cl.Do(req)
		if err != nil {
			t.Fatalf("Err received on %s->%s: %+v", req.Method, req.URL.String(), err)
		}

		test.OUT.assertResponse(t, resp)
	}
}
