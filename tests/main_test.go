//go:build integration

package tests

import (
	"flag"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	flag.BoolVar(&Verbose, "verbose", false, "enable more verbose logging")
	flag.Parse()

	if st := m.Run(); st != 0 {
		os.Exit(st)
	}
}
