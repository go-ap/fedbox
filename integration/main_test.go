//go:build integration

package integration

import (
	"flag"
	"os"
	"testing"
)

var Verbose bool

func TestMain(m *testing.M) {
	flag.BoolVar(&Verbose, "verbose", false, "enable more verbose logging")
	flag.Parse()

	if st := m.Run(); st != 0 {
		os.Exit(st)
	}
}
