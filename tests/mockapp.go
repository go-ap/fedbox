package tests

import (
	"bytes"
	"fmt"
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/auth"
	"github.com/go-ap/fedbox/app"
	"github.com/go-ap/fedbox/cmd"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
	"github.com/go-ap/fedbox/internal/log"
	"github.com/go-ap/fedbox/storage/boltdb"
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

func addMockObjects(r storage.Repository, obj pub.ItemCollection, errFn app.LogFn) error {
	var err error
	for _, it := range obj {
		if it.GetLink() == "" {
			continue
		}
		if pub.ActivityTypes.Contains(it.GetType()) {
			it, err = r.SaveActivity(it)
		} else if pub.ActorTypes.Contains(it.GetType()) {
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

func seedTestData(t *testing.T, testData []string) {
	if t != nil {
		t.Helper()
		t.Logf("Resetting storage backend")
	}

	curPath, err := os.Getwd()
	if err != nil {
		curPath = os.TempDir()
	}
	u, _ := url.Parse(apiURL)
	b, s := getBoldDBs(curPath, u, "test", logrus.New())

	o := cmd.New(s, b, config.Options{})

	pw := []byte("hahah")
	defaultTestApp.Id, _ = o.AddClient(pw, []string{authCallbackURL}, nil)

	if len(testData) == 0 {
		return
	}
	mocks := make(pub.ItemCollection, 0)
	for _, path := range testData {
		json := loadMockJson(path, nil)()
		if json == "" {
			continue
		}
		it, err := pub.UnmarshalJSON([]byte(json))
		if err == nil {
			mocks = append(mocks, it)
		}
	}
	addMockObjects(o.Storage, mocks, t.Logf)

	tok, err := o.GenAuthToken(defaultTestApp.Id, defaultTestAccount.Id, nil)
	if err == nil {
		defaultTestAccount.AuthToken = tok
	}
}

const boltdbExt = "bdb"

func resetDB(t *testing.T) string {
	if t != nil {
		t.Helper()
		t.Logf("Resetting storage backend")
	}

	curPath, err := os.Getwd()
	if err != nil {
		curPath = os.TempDir()
	}
	dbPath := config.GetDBPath(curPath, host, "test")
	boltdb.Clean(dbPath)
	boltdb.Bootstrap(dbPath, apiURL)
	return dbPath
}

func getBoldDBs(dir string, u *url.URL, env env.Type, l logrus.FieldLogger) (storage.Repository, osin.Storage) {
	path := config.GetDBPath(dir, host, env)
	b := boltdb.New(boltdb.Config{
		Path:  path,
		LogFn: app.InfoLogFn(l),
		ErrFn: app.ErrLogFn(l),
	}, u.String())

	pathOauth := config.GetDBPath(dir, fmt.Sprintf("%s-oauth", host), env)
	if _, err := os.Stat(pathOauth); os.IsNotExist(err) {
		err := auth.BootstrapBoltDB(pathOauth, []byte(host))
		if err != nil {
			l.Errorf("Unable to create missing boltdb file %s: %s", pathOauth, err)
		}
	}

	s := auth.NewBoltDBStore(auth.BoltConfig{
		Path:       pathOauth,
		BucketName: host,
		LogFn:      app.InfoLogFn(l),
		ErrFn:      app.ErrLogFn(l),
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

	a, _ := app.New(l, "HEAD", string(e))
	r := chi.NewRouter()
	a.Storage = b
	_oauthServer, _ = osinServer(s, l)

	r.Use(app.Repo(b))
	r.Use(log.NewStructuredLogger(l))
	r.Route("/", a.Routes(a.Config().BaseURL, _oauthServer, l))

	return a.Run(r, time.Second)
}
