package integration

import (
	"context"
	"crypto/tls"
	"net/http"
	"path/filepath"
	"testing"

	containers "github.com/testcontainers/testcontainers-go"
)

var defaultFedBOXImage = "localhost/fedbox/app:dev"

var httpClient = http.Client{
	Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
}

func Test_Fetch(t *testing.T) {
	storage := filepath.Join(".", "mocks", "c2s")
	env := filepath.Join(storage, ".env")
	ctx := context.Background()
	fc, err := Run(ctx, defaultFedBOXImage, WithEnvFile(env), WithStorage(storage))
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	defer containers.CleanupContainer(t, fc)

	req, err := fc.Req(ctx, http.MethodGet, "https://fedbox", nil)
	if err != nil {
		t.Fatalf("unable to get container request: %s", err)
	}
	r, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("Err received: %s", err)
	}

	if r.StatusCode != http.StatusOK {
		t.Errorf("Invalid status received %d, expected %d", r.StatusCode, http.StatusOK)
	}
}
