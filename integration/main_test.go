package integration

import (
	"context"
	"flag"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
)

var (
	Verbose bool
	Build   bool

	FedBOXImageName      = "localhost/fedbox/app:dev"
	defaultAuthImageName = "localhost/auth/app:dev"
)

func TestMain(m *testing.M) {
	flag.BoolVar(&Verbose, "verbose", false, "enable more verbose logging")
	flag.BoolVar(&Build, "build", false, "build images before run")
	flag.StringVar(&FedBOXImageName, "image", "quay.io/go-ap/fedbox:latest", "which image to use for running the tests")
	flag.Parse()

	if Build {
		logger := logrus.New()
		logger.SetOutput(os.Stderr)
		if Verbose {
			logger.SetLevel(logrus.DebugLevel)
		}
		logger.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true, DisableQuote: true, ForceColors: true, DisableLevelTruncation: true})

		name, err := buildImage(context.Background(), FedBOXImageName, logger)
		if err != nil {
			logger.Errorf("error building: %+v", err)
		} else {
			FedBOXImageName = name
			logger.Infof("built image: %s", name)
		}
	}

	if st := m.Run(); st != 0 {
		os.Exit(st)
	}
}
