package tests

import (
	"fmt"
	"github.com/go-ap/activitypub/client"
	"github.com/go-ap/auth"
	"github.com/go-ap/fedbox/app"
	"github.com/go-ap/fedbox/internal/log"
	"github.com/go-ap/fedbox/storage/boltdb"
	"github.com/go-ap/fedbox/validation"
	"github.com/go-chi/chi"
	"github.com/openshift/osin"
	"github.com/sirupsen/logrus"
	"os"
	"testing"
	"time"
)

func resetDB(t *testing.T, testData bool) string {
	if t != nil {
		t.Helper()
		t.Logf("Resetting DB")
	}

	curPath, err := os.Getwd()
	if err != nil {
		curPath = os.TempDir()
	}
	path := fmt.Sprintf("%s/%s-%d.bdb", curPath, "test", os.Getpid())

	boltdb.Clean(path, []byte(host))
	boltdb.Bootstrap(path, []byte(host), apiURL)

	return path
}

func runAPP(e string) int {
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)

	curPath, err := os.Getwd()
	if err != nil {
		curPath = os.TempDir()
	}
	path := fmt.Sprintf("%s/%s-%d.bdb", curPath, "test", os.Getpid())
	pathOauth := fmt.Sprintf("%s/%s-%d-oauth.bdb", curPath, "test", os.Getpid())

	b, err := boltdb.New(boltdb.Config{
		Path:       path,
		BucketName: host,
		LogFn:      func(f logrus.Fields, s string, p ...interface{}) { l.Errorf(s, p...) },
		ErrFn:      func(f logrus.Fields, s string, p ...interface{}) { l.Infof(s, p...) },
	}, apiURL)
	if err != nil {
		return 1
	}
	defer b.Close()

	a := app.New(l, "HEAD", e)
	r := chi.NewRouter()
	cl := osin.DefaultClient{
		Id:          "default",
		Secret:      "hahah",
		RedirectUri: authCallbackURL,
		UserData:    nil,
	}
	auth.BootstrapBoltDB(pathOauth, []byte(host), &cl)

	oStor := auth.NewBoltDBStore(auth.Config{
		Path:       pathOauth,
		BucketName: host,
		LogFn:      func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Infof(s, p...) },
		ErrFn:      func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Errorf(s, p...) },
	})

	defer oStor.Close()
	_oauthServer, _ = osinServer(oStor, l)

	r.Use(app.Repo(b))
	r.Use(log.NewStructuredLogger(l))
	r.Route("/", app.Routes(validation.New(a.Config().BaseURL, client.NewClient(), b), _oauthServer, b, l))

	return a.Run(r, time.Second)
}
