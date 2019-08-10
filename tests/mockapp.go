package tests

import (
	"fmt"
	"github.com/go-ap/activitypub/client"
	"github.com/go-ap/auth"
	"github.com/go-ap/fedbox/app"
	"github.com/go-ap/fedbox/cmd"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/log"
	"github.com/go-ap/fedbox/storage/boltdb"
	"github.com/go-ap/fedbox/validation"
	"github.com/go-chi/chi"
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
	path := config.GetBoltDBPath(curPath, fmt.Sprintf("%s-%d", host, os.Getpid()), "test")

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
	path := config.GetBoltDBPath(curPath, fmt.Sprintf("%s-%d", host, os.Getpid()), "test")
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
	pathOauth := config.GetBoltDBPath(curPath, fmt.Sprintf("%s-oauth-%d", host, os.Getpid()), "test")
	if _, err := os.Stat(pathOauth); os.IsNotExist(err) {
		err := auth.BootstrapBoltDB(pathOauth, []byte(host))
		if err != nil {
			l.Errorf("Unable to create missing boltdb file %s: %s", pathOauth, err)
		}
	}

	oStor := auth.NewBoltDBStore(auth.Config{
		Path:       pathOauth,
		BucketName: host,
		LogFn:      func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Infof(s, p...) },
		ErrFn:      func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Errorf(s, p...) },
	})
	defer oStor.Close()
	o := cmd.OauthCLI{
		AuthDB:  oStor,
		ActorDB: b,
	}
	pw := "hahah"
	id, _ := o.AddClient(pw, []string{authCallbackURL}, nil)

	defaultTestAccount.authToken, _ = o.GenAuthToken(id, defaultTestAccount.Handle, nil)
	_oauthServer, _ = osinServer(oStor, l)

	r.Use(app.Repo(b))
	r.Use(log.NewStructuredLogger(l))
	r.Route("/", app.Routes(validation.New(a.Config().BaseURL, client.NewClient(), b), _oauthServer, b, l))

	return a.Run(r, time.Second)
}
