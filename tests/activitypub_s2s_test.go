package tests

import (
	"fmt"
	"testing"
)

var selfAccount = testAccount{
	id:     fmt.Sprintf("http://%s/actors/d3ab037c-0f15-4c09-b635-3d6e201c11aa", host),
	Hash:   "d3ab037c-0f15-4c09-b635-3d6e201c11aa",
	Handle: "self",
}

var S2STests = testPairs{}

func Test_S2SRequests(t *testing.T) {
	testSuite(t, S2STests)
}
