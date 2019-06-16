package tests

import (
	"fmt"
	"github.com/go-ap/fedbox/app"
	"github.com/go-ap/fedbox/internal/log"
	"github.com/go-ap/fedbox/storage/boltdb"
	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
	"os"
	"testing"
	"time"
)

func resetDB(t *testing.T, testData bool) func() {
	if t != nil {
		t.Helper()
		t.Logf("Resetting DB")
	}

	curPath, err := os.Getwd()
	if err != nil {
		curPath = os.TempDir()
	}
	path := fmt.Sprintf("%s/%s-%d.bdb", curPath, "test", os.Getpid())
	rm := func() {
		err := os.Remove(path)
		if t != nil && err == nil {
			t.Logf("Removed db file: %s", path)
		}
	}
	rm()

	boltdb.Bootstrap(path, []byte(host), apiURL)
	return rm
}

func runAPP(e string) int {
	l := logrus.New()
	l.SetLevel(logrus.TraceLevel)

	def := resetDB(nil, true)
	curPath, err := os.Getwd()
	if err != nil {
		curPath = os.TempDir()
	}
	path := fmt.Sprintf("%s/%s-%d.bdb", curPath, "test", os.Getpid())

	b, err := boltdb.New(boltdb.Config{
		Path:       path,
		BucketName: host,
		ErrFn:      l.Errorf,
		LogFn:      l.Infof,
	}, apiURL)
	defer func() {
		b.Close()
		def()
	}()
	if err != nil {
		return 1
	}
	a := app.New(l, "HEAD", e)
	r := chi.NewRouter()

	r.Use(app.Repo(b))
	r.Use(log.NewStructuredLogger(l))
	r.Route("/", app.Routes())

	return a.Run(r, time.Second*5)
}
