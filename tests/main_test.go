package tests

import (
	"flag"
	"github.com/go-ap/fedbox/internal/env"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	var environ string

	flag.StringVar(&environ, "env", string(env.TEST), "environment")
	flag.Parse()

	go runAPP(env.Type(environ))

	time.Sleep(100 * time.Millisecond)

	st := m.Run()
	if st != 0 {
		os.Exit(st)
	}
}
