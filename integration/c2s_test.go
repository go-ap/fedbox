package integration

import (
	"context"
	"github.com/go-ap/client"
	containers "github.com/testcontainers/testcontainers-go"
	"net/http"
	"path/filepath"
	"testing"
)

var defaultFedBOXImage = "localhost/fedbox/app:dev"

func Test_Fetch(t *testing.T) {
	storage := filepath.Join(".", "mocks", "c2s")
	env := filepath.Join(storage, ".env")
	ctx := context.Background()
	fc, err := Run(ctx, defaultFedBOXImage, WithEnvFile(env), WithStorage(storage))
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	defer containers.CleanupContainer(t, fc)

	host, err := fc.Endpoint(ctx, "https")
	if err != nil {
		t.Fatalf("error: %s", err)
	}

	cl := client.New(client.SkipTLSValidation(true))

	r, err := cl.Get(host)
	if err != nil {
		t.Fatalf("Err received: %s", err)
	}
	//time.Sleep(time.Minute)
	if r.StatusCode != http.StatusOK {
		t.Errorf("Invalid status received %d, expected %d", r.StatusCode, http.StatusOK)
	}
}
