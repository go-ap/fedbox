// +build integration

package tests

import (
	"flag"
	"os"
	"testing"
	"time"

	"github.com/go-ap/fedbox/internal/env"
)

func TestMain(m *testing.M) {
	var environ string

	flag.StringVar(&environ, "env", string(env.TEST), "environment")
	flag.Parse()

	fedboxApp = SetupAPP(env.TEST)
	go fedboxApp.Run()

	time.Sleep(100 * time.Millisecond)

	st := m.Run()
	if st != 0 {
		os.Exit(st)
	}
}
