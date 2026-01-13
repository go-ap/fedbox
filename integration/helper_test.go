package integration

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/client"
	"github.com/go-ap/jsonld"
	"github.com/google/go-cmp/cmp"
)

type input struct {
	met     string
	headers http.Header
	urlFn   func() string
	body    io.Reader
}

func (req *input) Request(ctx context.Context, mocks cntrs) (*http.Request, error) {
	return mocks.Req(ctx, req.met, req.urlFn(), nil)
}

type reqInitFn func(*input)

func withURL(u string) reqInitFn {
	return func(t *input) {
		t.urlFn = func() string {
			return u
		}
	}
}

func withMethod(m string) reqInitFn {
	return func(t *input) {
		t.met = m
	}
}

func withBody(r io.Reader) reqInitFn {
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

func in(initFn ...reqInitFn) input {
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

type output struct {
	code int
	it   vocab.Item
}

type resInitFn func(*output)

func hasCode(c int) resInitFn {
	return func(res *output) {
		res.code = c
	}
}
func hasItem(it vocab.Item) resInitFn {
	return func(res *output) {
		res.it = it
	}
}

func out(initFn ...resInitFn) output {
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

type ioTest struct {
	name   string
	input  input
	output output
}

func (test ioTest) run(ctx context.Context, mocks cntrs) func(t *testing.T) {
	cl := client.New(client.WithHTTPClient(&httpClient), client.SkipTLSValidation(true))
	return func(t *testing.T) {
		req, err := test.input.Request(ctx, mocks)
		if err != nil {
			t.Fatalf("unable to create request: %+v", err)
		}

		resp, err := cl.Do(req)
		if err != nil {
			t.Fatalf("Err received: %+v", err)
		}

		test.output.assertResponse(t, resp)
	}
}
