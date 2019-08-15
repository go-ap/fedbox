package app

import (
	"github.com/go-ap/errors"
	"testing"
)

func TestNew(t *testing.T) {
	app, err := New(nil, "HEAD", "test")
	if err == nil {
		t.Errorf("Environment less should trigger an error")
	}
	if !errors.IsNotImplemented(err) {
		t.Errorf("Environment less should trigger a missing storage type error, received %T: %s", err, err)
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
