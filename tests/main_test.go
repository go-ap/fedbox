//go:build integration
// +build integration

package tests

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	if st := m.Run(); st != 0 {
		os.Exit(st)
	}
}
