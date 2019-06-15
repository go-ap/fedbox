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

	go runAPP(environ)

	time.Sleep(50 * time.Millisecond)
	os.Exit(m.Run())
}
