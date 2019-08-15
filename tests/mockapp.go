package tests

import (
	"fmt"
	"github.com/go-ap/activitypub/client"
	"github.com/go-ap/activitystreams"
	"github.com/go-ap/auth"
	"github.com/go-ap/fedbox/app"
	"github.com/go-ap/fedbox/cmd"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
	"github.com/go-ap/fedbox/internal/log"
	"github.com/go-ap/fedbox/storage/boltdb"
	"github.com/go-ap/fedbox/validation"
	"github.com/go-ap/storage"
	"github.com/go-chi/chi"
	"github.com/openshift/osin"
	"github.com/sirupsen/logrus"
	"net/url"
	"os"
	"path"
	"testing"
	"time"
)

func resetDB(t *testing.T, testData bool) string {
	if t != nil {
		t.Helper()
		t.Logf("Resetting storage backend")
	}

	curPath, err := os.Getwd()
	if err != nil {
		curPath = os.TempDir()
	}
	dbPath := config.GetBoltDBPath(curPath, host, "test")

	boltdb.Clean(dbPath, []byte(host))
	boltdb.Bootstrap(dbPath, []byte(host), apiURL)

	u, _ := url.Parse(apiURL)
	b, s := getBoldDBs(curPath, u, "test", logrus.New())

	o := cmd.New(u, s, b, config.Options{})

	pw := []byte("hahah")
	actor, err := o.AddActor(testActorHandle, activitystreams.PersonType, pw)
	if err == nil {
		defaultTestAccount.id = actor.GetLink().String()
		defaultTestAccount.Hash = path.Base(defaultTestAccount.id)
	}

	id, _ := o.AddClient(pw, []string{authCallbackURL}, nil)
	tok, err := o.GenAuthToken(id, defaultTestAccount.id, nil)
	if err == nil {
		defaultTestAccount.authToken = tok
	}

	return dbPath
}

func getBoldDBs(dir string, u *url.URL, env env.Type, l logrus.FieldLogger) (storage.Repository, osin.Storage) {
	path := config.GetBoltDBPath(dir, host, env)
	b := boltdb.New(boltdb.Config{
		Path:       path,
		BucketName: host,
		LogFn:      func(f logrus.Fields, s string, p ...interface{}) { l.Errorf(s, p...) },
		ErrFn:      func(f logrus.Fields, s string, p ...interface{}) { l.Infof(s, p...) },
	}, u.String())

	pathOauth := config.GetBoltDBPath(dir, fmt.Sprintf("%s-oauth", host), env)
	if _, err := os.Stat(pathOauth); os.IsNotExist(err) {
		err := auth.BootstrapBoltDB(pathOauth, []byte(host))
		if err != nil {
			l.Errorf("Unable to create missing boltdb file %s: %s", pathOauth, err)
		}
	}

	s := auth.NewBoltDBStore(auth.BoltConfig{
		Path:       pathOauth,
		BucketName: host,
		LogFn:      func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Infof(s, p...) },
		ErrFn:      func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Errorf(s, p...) },
	})

	return b, s
}

func runAPP(e env.Type) int {
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)

	curPath, err := os.Getwd()
	if err != nil {
		curPath = os.TempDir()
	}

	u, _ := url.Parse(apiURL)
	b, s := getBoldDBs(curPath, u, "test", l)

	a, _ := app.New(l, "HEAD", e)
	r := chi.NewRouter()

	_oauthServer, _ = osinServer(s, l)

	r.Use(app.Repo(b))
	r.Use(log.NewStructuredLogger(l))
	r.Route("/", app.Routes(validation.New(a.Config().BaseURL, client.NewClient(), b), _oauthServer, b, l))

	return a.Run(r, time.Second)
}
