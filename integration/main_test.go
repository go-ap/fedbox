package integration

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"
)

var Verbose bool
var Build bool

func TestMain(m *testing.M) {
	flag.BoolVar(&Verbose, "verbose", false, "enable more verbose logging")
	flag.BoolVar(&Build, "build", false, "build images before run")
	flag.Parse()

	if Build {
		name, err := buildImage(context.Background())
		if err != nil {
			fmt.Fprintf(os.Stderr, "error building: %s", err)
			os.Exit(-1)
		}
		fmt.Fprintf(os.Stdout, "built image: %s", name)
	}

	if st := m.Run(); st != 0 {
		os.Exit(st)
	}
}
