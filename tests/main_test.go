// +build integration

package tests

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	st := m.Run()
	if st != 0 {
		os.Exit(st)
	}
}
