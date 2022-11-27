package fedbox

import (
	"testing"

	"github.com/go-ap/fedbox/internal/config"
	fs "github.com/go-ap/storage-fs"
)

var defaultConfig = config.Options{
	Storage: config.StorageFS,
}

func TestNew(t *testing.T) {
	store, err := fs.New(fs.Config{Path: t.TempDir()})
	if err != nil {
		t.Errorf("unable to initialize fs storage: %s", err)
	}
	app, err := New(nil, "HEAD", config.Options{BaseURL: "http://example.com"}, store, nil)
	if err != nil {
		t.Errorf("Environment 'test' should not trigger an error: %s", err)
	}
	if app == nil {
		t.Errorf("Nil app pointer returned by New")
	}
}

func TestFedbox_Config(t *testing.T) {
	t.Skipf("TODO")
}

func TestFedbox_Run(t *testing.T) {
	t.Skipf("TODO")
}

func TestFedbox_Stop(t *testing.T) {
	t.Skipf("TODO")
}
