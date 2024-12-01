package integration

import (
	"context"
	"crypto/tls"
	"net/http"
	"testing"
	"time"
)

var httpClient = http.Client{
	Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
}

func Test_Fetch(t *testing.T) {
	ctx := context.Background()

	mocks, err := initMocks(ctx, "fedbox")
	defer mocks.cleanup(t)

	if err != nil {
		t.Fatalf("unable to initialize containers: %s", err)
	}

	req, err := mocks.Req(ctx, http.MethodGet, "https://fedbox", nil)
	r, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("Err received: %s", err)
	}

	if r.StatusCode != http.StatusOK {
		time.Sleep(2 * time.Minute)
		t.Errorf("Invalid status received %d, expected %d", r.StatusCode, http.StatusOK)
	}
}
