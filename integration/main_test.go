package integration

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"

	"git.sr.ht/~mariusor/storage-all"
	"github.com/go-ap/fedbox/integration/internal/containers"
	"github.com/sirupsen/logrus"
)

var (
	Build   bool
	Verbose bool
	Storage string

	fedBOXImageName      = "localhost/fedbox/app"
	defaultAuthImageName = "localhost/auth/app"

	validStorageTypes = []string{
		"boltdb",
		"fs",
		"badger",
		"sqlite",
		"postgres",
	}
)

func TestMain(m *testing.M) {
	flag.BoolVar(&Verbose, "verbose", false, "enable more verbose logging")
	flag.BoolVar(&Build, "build", false, "build images before run")
	flag.StringVar(&Storage, "storage", string(storage.Default), fmt.Sprintf("which storage type to use for tests, valid values: %#v", validStorageTypes))
	flag.Parse()

	logger := logrus.New()
	logger.SetOutput(os.Stderr)
	if Verbose {
		logger.SetLevel(logrus.TraceLevel)
		defaultC2SEnv["LOG_LEVEL"] = "trace"
	}
	logger.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true, DisableQuote: true, ForceColors: true, DisableLevelTruncation: true})

	if Build {
		name, err := containers.BuildImage(context.Background(), fedBOXImageName, logger)
		if err != nil {
			logger.Fatalf("error building: %+v", err)
		} else {
			fedBOXImageName = name
			logger.Infof("built image: %s", name)
		}
	}
	if st := m.Run(); st != 0 {
		os.Exit(st)
	}
}
