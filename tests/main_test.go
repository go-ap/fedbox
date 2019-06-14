package tests

import (
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	go runAPP()
	
	time.Sleep(50 * time.Millisecond)
	os.Exit(m.Run())
}
