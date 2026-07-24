package tests

import (
	"context"
	"crypto/tls"
	"net/http"
	"testing"
	"time"

	c "github.com/go-ap/fedbox/integration/internal/containers"
)

type TestSuite struct {
	Name  string
	Tests []RunnableTest
}

func (ht TestSuite) Label() string {
	return ht.Name
}

func (ht TestSuite) Fn(ctx context.Context, mocks c.Running) func(t *testing.T) {
	return func(t *testing.T) {
		ht.Run(ctx, mocks, t)
	}
}

func (ht TestSuite) Run(ctx context.Context, mocks c.Running, t *testing.T) {
	for _, tt := range ht.Tests {
		t.Run(tt.Label(), func(t *testing.T) {
			tt.Run(ctx, mocks, t)
		})
	}
}

type HTTPTest struct {
	Name string
	Req  *reqBuilder
	Res  resChecks
}

func (ht HTTPTest) Label() string {
	return ht.Name
}

func (ht HTTPTest) Fn(ctx context.Context, mocks c.Running) func(t *testing.T) {
	return func(t *testing.T) {
		ht.Run(ctx, mocks, t)
	}
}

func (ht HTTPTest) Run(ctx context.Context, mocks c.Running, t *testing.T) {
	cl := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       20 * time.Second,
			TLSHandshakeTimeout:   2 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			DialContext:           mocks.DialContext,
		},
	}

	var cancelFn func()
	ctx, cancelFn = context.WithTimeout(ctx, 10*time.Second)
	defer cancelFn()

	req, err := ht.Req.Request(ctx)
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
