package fedbox

import (
	"testing"

	"github.com/go-ap/fedbox/internal/config"
)

var defaultConfig = config.Options{
	Storage: config.StorageFS,
}

func TestNew(t *testing.T) {
	app, err := New(nil, "HEAD", defaultConfig, nil, nil)
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
