package tests

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/carlmjohnson/requests"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/client"
)

type reqBuilder struct {
	*requests.Builder
	signer func(*http.Request) error
}

func Request() *reqBuilder {
	b := reqBuilder{Builder: requests.New()}
	return &b
}

func (rb *reqBuilder) URL(s string) *reqBuilder {
	rb.Builder = rb.Builder.BaseURL(s)
	if u, err := url.Parse(s); err == nil {
		rb.Builder = rb.Builder.Header("Host", u.Hostname())
	}
	return rb
}

func (rb *reqBuilder) IRI(s vocab.IRI) *reqBuilder {
	return rb.URL(string(s))
}

func (rb *reqBuilder) Method(method string) *reqBuilder {
	rb.Builder = rb.Builder.Method(method)
	return rb
}

func (rb *reqBuilder) Head() *reqBuilder {
	rb.Builder = rb.Builder.Head()
	return rb
}

func (rb *reqBuilder) Post() *reqBuilder {
	rb.Builder = rb.Builder.Post()
	return rb
}

func (rb *reqBuilder) Hostf(format string, a ...any) *reqBuilder {
	rb.Builder = rb.Builder.Hostf(format, a...)
	return rb
}

func (rb *reqBuilder) Pathf(format string, a ...any) *reqBuilder {
	rb.Builder = rb.Builder.Pathf(format, a...)
	return rb
}

func (rb *reqBuilder) ParamInt(key string, value int) *reqBuilder {
	rb.Builder = rb.Builder.ParamInt(key, value)
	return rb
}

func (rb *reqBuilder) Params(m map[string][]string) *reqBuilder {
	rb.Builder = rb.Builder.Params(m)
	return rb
}

func (rb *reqBuilder) Header(k string, v ...string) *reqBuilder {
	rb.Builder = rb.Builder.Header(k, v...)
	return rb
}

func (rb *reqBuilder) Headers(m map[string][]string) *reqBuilder {
	rb.Builder = rb.Builder.Headers(m)
	return rb
}

func (rb *reqBuilder) Signer(signFn func(*http.Request) error) *reqBuilder {
	rb.signer = signFn
	return rb
}

func (rb *reqBuilder) Accept(contentTypes string) *reqBuilder {
	rb.Builder = rb.Builder.Accept(contentTypes)
	return rb
}

func (rb *reqBuilder) CacheControl(directive string) *reqBuilder {
	rb.Builder = rb.Builder.CacheControl(directive)
	return rb
}

func (rb *reqBuilder) ContentType(ct string) *reqBuilder {
	rb.Builder = rb.Builder.ContentType(ct)
	return rb
}

func (rb *reqBuilder) UserAgent(s string) *reqBuilder {
	rb.Builder = rb.Builder.UserAgent(s)
	return rb
}

func (rb *reqBuilder) BasicAuth(username, password string) *reqBuilder {
	rb.Builder = rb.Builder.BasicAuth(username, password)
	return rb
}

func (rb *reqBuilder) Bearer(token string) *reqBuilder {
	rb.Builder = rb.Builder.Bearer(token)
	return rb
}

func (rb *reqBuilder) BodyReader(r io.Reader) *reqBuilder {
	rb.Builder = rb.Builder.Body(requests.BodyReader(r))
	return rb
}

func (rb *reqBuilder) BodyWriter(f func(w io.Writer) error) *reqBuilder {
	rb.Builder = rb.Builder.BodyWriter(f)
	return rb
}

func (rb *reqBuilder) BodyItem(it vocab.Item) *reqBuilder {
	raw, err := vocab.MarshalJSON(it)
	if err != nil {
		panic(fmt.Errorf("unable to marshal item: %v", err))
	}
	rb.Builder = rb.Builder.ContentType(client.ContentTypeJsonLD).BodyBytes(raw)
	return rb
}

func (rb *reqBuilder) BodyBytes(b []byte) *reqBuilder {
	rb.Builder = rb.Builder.BodyBytes(b)
	return rb
}

func (rb *reqBuilder) BodySerializer(s requests.Serializer, v any) *reqBuilder {
	rb.Builder = rb.Builder.BodySerializer(s, v)
	return rb
}

func (rb *reqBuilder) BodyJSON(v any) *reqBuilder {
	rb.Builder = rb.Builder.BodyJSON(v)
	return rb
}

func (rb *reqBuilder) BodyForm(data url.Values) *reqBuilder {
	rb.Builder = rb.Builder.BodyForm(data)
	return rb
}

func (rb *reqBuilder) BodyFile(name string) *reqBuilder {
	rb.Builder = rb.Builder.BodyFile(name)
	return rb
}

func (rb *reqBuilder) CheckStatus(acceptStatuses ...int) *reqBuilder {
	rb.Builder = rb.Builder.CheckStatus(acceptStatuses...)
	return rb
}

func (rb *reqBuilder) CheckContentType(cts ...string) *reqBuilder {
	rb.Builder = rb.Builder.CheckContentType(cts...)
	return rb
}

func (rb *reqBuilder) ToJSON(v any) *reqBuilder {
	rb.Builder = rb.Builder.ToJSON(v)
	return rb
}

func (rb *reqBuilder) ToString(sp *string) *reqBuilder {
	rb.Builder = rb.Builder.ToString(sp)
	return rb
}

func (rb *reqBuilder) ToBytesBuffer(buf *bytes.Buffer) *reqBuilder {
	rb.Builder = rb.Builder.ToBytesBuffer(buf)
	return rb
}

func (rb *reqBuilder) ToWriter(w io.Writer) *reqBuilder {
	rb.Builder = rb.Builder.ToWriter(w)
	return rb
}

func (rb *reqBuilder) CopyHeaders(h map[string][]string) *reqBuilder {
	rb.Builder = rb.Builder.CopyHeaders(h)
	return rb
}

func (rb *reqBuilder) ToHeaders(h map[string][]string) *reqBuilder {
	rb.Builder = rb.Builder.ToHeaders(h)
	return rb
}

func (rb *reqBuilder) ErrorJSON(v any) *reqBuilder {
	rb.Builder = rb.Builder.ErrorJSON(v)
	return rb
}

// redirecter is the interface for modifying a mocked request to reach the correct server in a
// group of test containers.
type redirecter interface {
	RedirectRequest(context.Context, *http.Request) error
}

func (rb *reqBuilder) Request(ctx context.Context) (*http.Request, error) {
	if rb == nil {
		return nil, fmt.Errorf("nil request")
	}
	req, err := rb.Builder.Request(ctx)
	if err != nil {
		return nil, err
	}

	if rb.signer != nil {
		if err = rb.signer(req); err != nil {
			return req, err
		}
	}
	return req, nil
}
