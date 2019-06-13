package tests

import (
	"fmt"
	"github.com/go-ap/storage/boltdb"
	log "github.com/sirupsen/logrus"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	l := log.New()
	l.SetLevel(log.TraceLevel)

	resetDB(nil, true)
	curPath, err := os.Getwd()
	if err != nil {
		curPath = os.TempDir()
	}
	path := fmt.Sprintf("%s/%s-%d.bdb", curPath, "test", os.Getpid())

	if b, err := boltdb.New(boltdb.Config{
		Path:       path,
		BucketName: host,
		ErrFn:      l.Errorf,
		LogFn:      l.Infof,
	});  err == nil {
		defer b.Close()
		go runAPP(l, b)
	} else {
		l.Errorf("Unable to open bolt db: %s", err)
	}
	defer os.Remove(path)


	os.Exit(m.Run())
}
