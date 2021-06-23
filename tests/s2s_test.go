// +build integration,s2s

package tests

import (
	"testing"
)

var S2STests = testPairs{}

func Test_S2SRequests(t *testing.T) {
	runTestSuite(t, S2STests)
}
