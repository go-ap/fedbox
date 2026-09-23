package integration

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"

	st "git.sr.ht/~mariusor/storage-all"
	"github.com/go-ap/fedbox/integration/internal/containers"
)

var (
	build    bool
	verbose  bool
	race     bool
	coverage bool
	storage  string

	imageName = "localhost/fedbox/app"

	validStorageTypes = []string{
		string(st.FS),       // fs
		string(st.Badger),   // badger
		string(st.BoltDB),   // boltdb
		string(st.Sqlite),   // sqlite
		string(st.Postgres), // postgres
	}
)

func TestMain(m *testing.M) {
	name := imageName

	flag.BoolVar(&verbose, "verbose", false, "enable more verbose logging")
	flag.BoolVar(&race, "race", false, "build the image with data race detection")
	flag.BoolVar(&coverage, "coverage", false, "build the image with test coverage support")
	flag.BoolVar(&build, "build", false, "build images before run")
	flag.StringVar(&name, "name", imageName, "which container image to use")
	flag.StringVar(&storage, "storage", string(st.Default), fmt.Sprintf("which storage type to use for tests, valid values: %#v", validStorageTypes))
	flag.Parse()

	if build {
		var err error
		if name, err = containers.BuildImage(context.Background(), imageName, coverage, race); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "error building image: %+v", err)
			os.Exit(-1)
		}
		imageName = name
		_, _ = fmt.Fprintf(os.Stderr, "built image: %s", name)
	}
	if st := m.Run(); st != 0 {
		os.Exit(st)
	}
}
