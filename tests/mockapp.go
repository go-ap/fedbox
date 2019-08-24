package tests

import (
	"bytes"
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
	"io"
	"net/url"
	"os"
	"path"
	"testing"
	"text/template"
	"time"
)

func loadMockJson(file string, model interface{}) func() string {
	f, err := os.Open(file)
	if err != nil {
		return func() string { return "" }
	}

	st, err := f.Stat()
	if err != nil {
		return func() string { return "" }
	}

	data := make([]byte, st.Size())
	io.ReadFull(f, data)
	data = bytes.Trim(data, "\x00")

	t := template.Must(template.New(fmt.Sprintf("mock_%s", path.Base(file))).Parse(string(data)))

	return func() string {
		bytes := bytes.Buffer{}
		t.Execute(&bytes, model)
		return bytes.String()
	}
}

func addMockObjects(r storage.Repository, obj activitystreams.ItemCollection, errFn app.LogFn) error {
	var err error
	for _, it := range obj {
		if it.GetLink() == "" {
			continue
		}
		if activitystreams.ActivityTypes.Contains(it.GetType()) {
			it, err = r.SaveActivity(it)
		} else if activitystreams.ActorTypes.Contains(it.GetType()) {
			it, err = r.SaveActor(it)
		} else {
			it, err = r.SaveObject(it)
		}
		if err != nil {
			errFn("%s", err)
		}
	}
	return nil
}

func resetDB(t *testing.T, testData []string) string {
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

	mocks := make(activitystreams.ItemCollection, 0)
	for _, path := range testData {
		json := loadMockJson(path, nil)()
		if json == "" {
			continue
		}
		it, err := activitystreams.UnmarshalJSON([]byte(json))
		if err == nil {
			mocks = append(mocks, it)
		}
	}
	addMockObjects(o.Storage, mocks, t.Logf)

	pw := []byte("hahah")
	actor, err := o.AddActor(testActorHandle, activitystreams.PersonType, pw)
	if err == nil {
		defaultTestAccount.Id = actor.GetLink().String()
		defaultTestAccount.Hash = path.Base(defaultTestAccount.Id)
	}

	id, _ := o.AddClient(pw, []string{authCallbackURL}, nil)
	tok, err := o.GenAuthToken(id, defaultTestAccount.Id, nil)
	if err == nil {
		defaultTestAccount.AuthToken = tok
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
