package integration

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"

	"git.sr.ht/~mariusor/storage-all"
	"github.com/go-ap/fedbox/integration/internal/containers"
)

var (
	Build   bool
	Verbose bool
	Race    bool
	Storage string

	fedBOXImageName = "localhost/fedbox/app"

	validStorageTypes = []string{
		string(storage.FS),       // fs
		string(storage.Badger),   // badger
		string(storage.BoltDB),   // boltdb
		string(storage.Sqlite),   // sqlite
		string(storage.Postgres), // postgres
	}
)

func TestMain(m *testing.M) {
	name := fedBOXImageName

	flag.BoolVar(&Verbose, "verbose", false, "enable more verbose logging")
	flag.BoolVar(&Race, "race", false, "build the image with data race detection")
	flag.BoolVar(&Build, "build", false, "build images before run")
	flag.StringVar(&name, "name", fedBOXImageName, "which container image to use")
	flag.StringVar(&Storage, "storage", string(storage.Default), fmt.Sprintf("which storage type to use for tests, valid values: %#v", validStorageTypes))
	flag.Parse()

	if Build {
		var err error
		if name, err = containers.BuildImage(context.Background(), fedBOXImageName, Race); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "error building image: %+v", err)
			os.Exit(-1)
		}
		fedBOXImageName = name
		_, _ = fmt.Fprintf(os.Stderr, "built image: %s", name)
	}
	if st := m.Run(); st != 0 {
		os.Exit(st)
	}
}
