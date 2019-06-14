package tests

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	go runAPP()

	os.Exit(m.Run())
}
