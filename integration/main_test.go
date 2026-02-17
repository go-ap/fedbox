package integration

import (
	"context"
	"flag"
	"os"
	"testing"

	"github.com/go-ap/fedbox/integration/internal/containers"
	"github.com/sirupsen/logrus"
)

var (
	Build bool

	fedBOXImageName      = "localhost/fedbox/app"
	defaultAuthImageName = "localhost/auth/app"
)

func TestMain(m *testing.M) {
	flag.BoolVar(&containers.Verbose, "verbose", false, "enable more verbose logging")
	flag.BoolVar(&Build, "build", false, "build images before run")
	flag.Parse()

	if Build {
		logger := logrus.New()
		logger.SetOutput(os.Stderr)
		if containers.Verbose {
			logger.SetLevel(logrus.TraceLevel)
			defaultC2SEnv["LOG_LEVEL"] = "trace"
		}
		logger.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true, DisableQuote: true, ForceColors: true, DisableLevelTruncation: true})

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
