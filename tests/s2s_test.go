package tests

import (
	"fmt"
	"testing"
)

const serviceHash = "d3ab037c-0f15-4c09-b635-3d6e201c11aa"

var selfAccount = testAccount{
	Id:     fmt.Sprintf("http://%s/", host),
	Hash:   serviceHash,
	Handle: "self",
}

var S2STests = testPairs{}

func Test_S2SRequests(t *testing.T) {
	runTestSuite(t, S2STests)
}
