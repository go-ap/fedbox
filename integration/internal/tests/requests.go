package tests

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"

	"github.com/carlmjohnson/requests"
)

type builder struct {
	*requests.Builder
	signer func(*http.Request) error
}

func URL[T ~string](s T) *builder {
	b := builder{Builder: requests.URL(string(s))}
	if u, err := b.Builder.URL(); err == nil {
		b.Header("Host", u.Hostname())
	}
	return &b
}

func (rb *builder) Method(method string) *builder {
	rb.Builder = rb.Builder.Method(method)
	return rb
}

func (rb *builder) Head() *builder {
	rb.Builder = rb.Builder.Head()
	return rb
}

func (rb *builder) Post() *builder {
	rb.Builder = rb.Builder.Post()
	return rb
}

func (rb *builder) Hostf(format string, a ...any) *builder {
	rb.Builder = rb.Builder.Hostf(format, a...)
	return rb
}

func (rb *builder) Pathf(format string, a ...any) *builder {
	rb.Builder = rb.Builder.Pathf(format, a...)
	return rb
}

func (rb *builder) ParamInt(key string, value int) *builder {
	rb.Builder = rb.Builder.ParamInt(key, value)
	return rb
}

func (rb *builder) Params(m map[string][]string) *builder {
	rb.Builder = rb.Builder.Params(m)
	return rb
}

func (rb *builder) Header(k string, v ...string) *builder {
	rb.Builder = rb.Builder.Header(k, v...)
	return rb
}

func (rb *builder) Headers(m map[string][]string) *builder {
	rb.Builder = rb.Builder.Headers(m)
	return rb
}

func (rb *builder) Signer(signFn func(*http.Request) error) *builder {
	rb.signer = signFn
	return rb
}

func (rb *builder) Accept(contentTypes string) *builder {
	rb.Builder = rb.Builder.Accept(contentTypes)
	return rb
}

func (rb *builder) CacheControl(directive string) *builder {
	rb.Builder = rb.Builder.CacheControl(directive)
	return rb
}

func (rb *builder) ContentType(ct string) *builder {
	rb.Builder = rb.Builder.ContentType(ct)
	return rb
}

func (rb *builder) UserAgent(s string) *builder {
	rb.Builder = rb.Builder.UserAgent(s)
	return rb
}

func (rb *builder) BasicAuth(username, password string) *builder {
	rb.Builder = rb.Builder.BasicAuth(username, password)
	return rb
}

func (rb *builder) Bearer(token string) *builder {
	rb.Builder = rb.Builder.Bearer(token)
	return rb
}

func (rb *builder) BodyReader(r io.Reader) *builder {
	rb.Builder = rb.Builder.Body(requests.BodyReader(r))
	return rb
}

func (rb *builder) BodyWriter(f func(w io.Writer) error) *builder {
	rb.Builder = rb.Builder.BodyWriter(f)
	return rb
}

func (rb *builder) BodyBytes(b []byte) *builder {
	rb.Builder = rb.Builder.BodyBytes(b)
	return rb
}

func (rb *builder) BodySerializer(s requests.Serializer, v any) *builder {
	rb.Builder = rb.Builder.BodySerializer(s, v)
	return rb
}

func (rb *builder) BodyJSON(v any) *builder {
	rb.Builder = rb.Builder.BodyJSON(v)
	return rb
}

func (rb *builder) BodyForm(data url.Values) *builder {
	rb.Builder = rb.Builder.BodyForm(data)
	return rb
}

func (rb *builder) BodyFile(name string) *builder {
	rb.Builder = rb.Builder.BodyFile(name)
	return rb
}

func (rb *builder) CheckStatus(acceptStatuses ...int) *builder {
	rb.Builder = rb.Builder.CheckStatus(acceptStatuses...)
	return rb
}

func (rb *builder) CheckContentType(cts ...string) *builder {
	rb.Builder = rb.Builder.CheckContentType(cts...)
	return rb
}

func (rb *builder) ToJSON(v any) *builder {
	rb.Builder = rb.Builder.ToJSON(v)
	return rb
}

func (rb *builder) ToString(sp *string) *builder {
	rb.Builder = rb.Builder.ToString(sp)
	return rb
}

func (rb *builder) ToBytesBuffer(buf *bytes.Buffer) *builder {
	rb.Builder = rb.Builder.ToBytesBuffer(buf)
	return rb
}

func (rb *builder) ToWriter(w io.Writer) *builder {
	rb.Builder = rb.Builder.ToWriter(w)
	return rb
}

func (rb *builder) CopyHeaders(h map[string][]string) *builder {
	rb.Builder = rb.Builder.CopyHeaders(h)
	return rb
}

func (rb *builder) ToHeaders(h map[string][]string) *builder {
	rb.Builder = rb.Builder.ToHeaders(h)
	return rb
}

func (rb *builder) ErrorJSON(v any) *builder {
	rb.Builder = rb.Builder.ErrorJSON(v)
	return rb
}

type redirecter interface {
	RedirectRequest(context.Context, *http.Request) error
}

func (rb *builder) Request(ctx context.Context, mocks redirecter) (*http.Request, error) {
	req, err := rb.Builder.Request(ctx)
	if err != nil {
		return nil, err
	}

	if rb.signer != nil {
		if err = rb.signer(req); err != nil {
			return req, err
		}
	}
	return req, mocks.RedirectRequest(ctx, req)
}
