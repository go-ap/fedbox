package fedbox

import (
	"testing"

	"git.sr.ht/~mariusor/lw"
	"git.sr.ht/~mariusor/storage-all"
	"github.com/go-ap/fedbox/internal/config"
)

func TestNew(t *testing.T) {
	store, err := storage.New(storage.WithPath(t.TempDir()))
	if err != nil {
		t.Errorf("unable to initialize fs storage: %s", err)
	}
	app, err := New(lw.Dev(), config.Options{BaseURL: "http://example.com"}, store)
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
