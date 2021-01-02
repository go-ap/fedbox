// +build integration

package tests

import (
	"flag"
	"github.com/go-ap/fedbox/internal/env"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	var environ string

	flag.StringVar(&environ, "env", string(env.TEST), "environment")
	flag.Parse()

	fedboxApp = SetupAPP(env.TEST)
	go fedboxApp.Run()
	defer fedboxApp.Stop()

	st := m.Run()
	if st != 0 {
		os.Exit(st)
	}
}
