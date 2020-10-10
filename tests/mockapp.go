// +build integration

package tests

import (
	"bytes"
	"fmt"
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/fedbox/app"
	"github.com/go-ap/fedbox/cmd"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
	"github.com/go-ap/fedbox/internal/log"
	"github.com/go-ap/storage"
	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"path"
	"testing"
	"text/template"
	"time"
)

func loadMockJson(file string, model interface{}) func() string {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return func() string { return "" }
	}
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

func seedTestData(t *testing.T, testData []string, reset bool) {
	if t == nil {
		panic("invalid test context")
	}
	t.Helper()

	opt, _ := config.LoadFromEnv("test")
	if opt.Storage == "all" {
		opt.Storage = config.StorageFS
	}
	if reset {
		resetDB(opt)
	}
	fields:= logrus.Fields{"action":"seeding", "storage": opt.Storage, "path": opt.StoragePath}
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	db, aDb, err := app.Storage(opt, l.WithFields(fields))
	if err != nil {
		panic(err)
	}

	o := cmd.New(aDb, db, config.Options{})
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

func runAPP(e env.Type) int {
	opt, _ := config.LoadFromEnv(env.TEST)
	if opt.Storage == "all" {
		opt.Storage = config.StorageFS
	}
	fields:= logrus.Fields{"action":"running", "storage": opt.Storage, "path": opt.StoragePath}
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)

	r := chi.NewRouter()
	r.Use(log.NewStructuredLogger(l))

	a, _ := app.New(l, "HEAD", string(e))
	db, aDb, err := app.Storage(opt, l.WithFields(fields))
	if err != nil {
		panic(err)
	}
	if db != nil {
		a.Storage = db
	}
	if aDb != nil {
		a.OAuthStorage = aDb
	}
	o := osinServer(aDb, l)

	r.Use(app.Repo(a.Storage))
	r.Route("/", a.Routes(a.Config().BaseURL, o, l))

	return a.Run(r, time.Second)
}
