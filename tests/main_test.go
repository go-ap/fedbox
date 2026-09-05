//go:build integration

package tests

import (
	"flag"
	"os"
	"testing"

	"go.uber.org/goleak"
)

var checkLeaks bool

func TestMain(m *testing.M) {
	flag.BoolVar(&Verbose, "verbose", false, "enable more verbose logging")
	flag.BoolVar(&checkLeaks, "check-leaks", false, "enable leak checking with goleak")
	flag.Parse()

	if checkLeaks {
		defer goleak.VerifyTestMain(m)
	}
	if st := m.Run(); st != 0 {
		os.Exit(st)
	}
}
