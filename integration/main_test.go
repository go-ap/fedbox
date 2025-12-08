package integration

import (
	"context"
	"flag"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
)

var Verbose bool
var Build bool

func TestMain(m *testing.M) {
	flag.BoolVar(&Verbose, "verbose", false, "enable more verbose logging")
	flag.BoolVar(&Build, "build", false, "build images before run")
	flag.Parse()

	if Build {
		logger := logrus.New()
		logger.SetOutput(os.Stderr)
		if Verbose {
			logger.SetLevel(logrus.DebugLevel)
		}
		logger.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true, DisableQuote: true, ForceColors: true, DisableLevelTruncation: true})

		name, err := buildImage(context.Background(), logger)
		if err != nil {
			logger.Errorf("error building: %+v", err)
		} else {
			logger.Infof("built image: %s", name)
		}
	}

	if st := m.Run(); st != 0 {
		os.Exit(st)
	}
}
