package integration

import (
	"context"
	"crypto/tls"
	"net/http"
	"testing"

	vocab "github.com/go-ap/activitypub"
)

var httpClient = http.Client{
	Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
}

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
			name:   "FedBOX root",
			arg:    "https://fedbox",
			wanted: wanted{status: http.StatusOK},
		},
		{
			name:   "FedBOX Admin",
			arg:    "https://fedbox/actors/1",
			wanted: wanted{status: http.StatusOK},
		},
	}

	ctx := context.Background()
	mocks, err := initMocks(ctx, "fedbox")
	if err != nil {
		t.Fatalf("unable to initialize containers: %s", err)
	}

	t.Cleanup(func() {
		mocks.cleanup(t)
	})

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req, err := mocks.Req(ctx, http.MethodGet, test.arg.String(), nil)
			r, err := httpClient.Do(req)
			if err != nil {
				t.Fatalf("Err received: %s", err)
			}

			if r.StatusCode != test.wanted.status {
				t.Errorf("Invalid status received %d, expected %d", r.StatusCode, http.StatusOK)
			}
		})
	}
}
