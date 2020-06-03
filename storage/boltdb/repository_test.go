package boltdb

import (
	"fmt"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
	"github.com/sirupsen/logrus"
	"os"
	"testing"
)

func TestNew(t *testing.T) {
	dir, _ := os.Getwd()
	name := "test.db"
	path := fmt.Sprintf("%s/%s", dir, name)
	url := "random-string-not-an-URL"

	conf := Config{
		Path:  path,
		LogFn: func(f logrus.Fields, s string, p ...interface{}) { t.Logf(s, p...) },
		ErrFn: func(f logrus.Fields, s string, p ...interface{}) { t.Errorf(s, p...) },
	}
	repo := New(conf, url)
	if repo == nil {
		t.Errorf("Nil result from opening boltdb %s", path)
	}
	if repo.path != path {
		t.Errorf("Wrong configured path %s, expected %s", repo.path, path)
	}
	if repo.baseURL != url {
		t.Errorf("Wrong configured base URL %s, expected %s", repo.baseURL, url)
	}
	if repo.d != nil {
		t.Errorf("Non nil boltdb from New")
	}
	if repo.errFn == nil {
		t.Errorf("Nil error log function, expected %T[%p]", t.Errorf, t.Errorf)
	}
	if repo.logFn == nil {
		t.Errorf("Nil log function, expected %T[%p]", t.Logf, t.Logf)
	}
}

func TestRepo_Open(t *testing.T) {
	dir := os.TempDir()
	url := "random-string-not-an-URL"
	c := config.Options{
		StoragePath: dir,
		Host:        "example.com",
		Env:         env.TEST,
		BaseURL:     url,
	}
	path := config.GetDBPath(c.StoragePath, c.Host, c.Env)
	err := Bootstrap(c)
	if err != nil {
		t.Errorf("Unable to bootstrap boltdb %s: %s", path, err)
	}
	defer os.Remove(path)
	conf := Config{
		Path: path,
	}
	repo := New(conf, url)
	err = repo.Open()
	if err != nil {
		t.Errorf("Unable to open boltdb %s: %s", path, err)
	}
	if repo.d == nil {
		t.Errorf("Nil %T for path %s", repo.d, path)
	}
}

func TestRepo_Close(t *testing.T) {
	dir := os.TempDir()
	url := "random-string-not-an-URL"
	c := config.Options{
		StoragePath: dir,
		Host:        "example.com",
		Env:         env.TEST,
		BaseURL:     url,
	}
	path := config.GetDBPath(c.StoragePath, c.Host, c.Env)
	err := Bootstrap(c)
	if err != nil {
		t.Errorf("Unable to bootstrap boltdb %s: %s", path, err)
	}
	defer os.Remove(path)

	conf := Config{
		Path: path,
	}
	repo := New(conf, url)
	err = repo.Open()
	if err != nil {
		t.Errorf("Unable to open boltdb %s: %s", path, err)
	}
	err = repo.Close()
	if err != nil {
		t.Errorf("Unable to close boltdb %s: %s", path, err)
	}
	os.Remove(path)
}

func TestRepo_GenerateID(t *testing.T) {
	t.Skipf("TODO")
}

func TestRepo_Load(t *testing.T) {
	t.Skipf("TODO")
}

func TestRepo_LoadActivities(t *testing.T) {
	t.Skipf("TODO")
}

func TestRepo_LoadActors(t *testing.T) {
	t.Skipf("TODO")
}

func TestRepo_LoadCollection(t *testing.T) {
	t.Skipf("TODO")
}

func TestRepo_LoadObjects(t *testing.T) {
	t.Skipf("TODO")
}

func TestRepo_UpdateObject(t *testing.T) {
	t.Skipf("TODO")
}

func TestRepo_DeleteObject(t *testing.T) {
	t.Skipf("TODO")
}

func TestRepo_SaveObject(t *testing.T) {
	t.Skipf("TODO")
}

func TestRepo_SaveActivity(t *testing.T) {
	t.Skipf("TODO")
}
