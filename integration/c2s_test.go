package integration

import (
	"context"
	"crypto/tls"
	"net/http"
	"testing"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/client"
	ap "github.com/go-ap/fedbox/activitypub"
)

var httpClient = http.Client{
	Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
}

var self = ap.Self

func Test_Fetch(t *testing.T) {
	type wanted struct {
		status int
		item   vocab.Item
	}
	tests := []struct {
		name   string
		arg    vocab.IRI
		wanted wanted
	}{
		{
			name: "FedBOX root",
			arg:  "http://fedbox",
			wanted: wanted{
				status: http.StatusOK,
				item:   self("https://fedbox"),
			},
		},
		{
			name:   "FedBOX Admin",
			arg:    "http://fedbox/actors/1",
			wanted: wanted{status: http.StatusOK},
		},
		{
			name:   "sysop tag",
			arg:    "http://fedbox/objects/0",
			wanted: wanted{status: http.StatusOK},
		},
		{
			name:   "object 1",
			arg:    "http://fedbox/objects/1",
			wanted: wanted{status: http.StatusOK},
		},
		{
			name:   "actor 2",
			arg:    "http://fedbox/actors/2",
			wanted: wanted{status: http.StatusOK},
		},
		{
			name:   "actors",
			arg:    "http://fedbox/actors",
			wanted: wanted{status: http.StatusOK},
		},
		{
			name:   "objects",
			arg:    "http://fedbox/objects",
			wanted: wanted{status: http.StatusOK},
		},
	}

	ctx := context.Background()
	mocks, err := initMocks(ctx, t, suite{name: "fedbox"})
	if err != nil {
		t.Fatalf("unable to initialize containers: %s", err)
	}

	t.Cleanup(func() {
		mocks.cleanup(t)
	})

	cl := client.New(client.WithHTTPClient(&httpClient), client.SkipTLSValidation(true))
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req, err := mocks.Req(ctx, http.MethodGet, string(test.arg), nil)
			r, err := cl.Do(req)
			if err != nil {
				t.Fatalf("Err received: %+v", err)
			}

			if r.StatusCode != test.wanted.status {
				t.Errorf("Invalid status received %d, expected %d", r.StatusCode, http.StatusOK)
			}
		})
	}
}
