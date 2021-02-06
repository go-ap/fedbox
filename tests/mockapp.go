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
	ls "github.com/go-ap/fedbox/storage"
	"github.com/go-ap/storage"
	"github.com/go-chi/chi"
	"github.com/openshift/osin"
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

func addMockObjects(r storage.Store, obj pub.ItemCollection, errFn app.LogFn) error {
	var err error
	for _, it := range obj {
		if it.GetLink() == "" {
			continue
		}
		if it, err = r.Save(it); err != nil {
			errFn(err.Error())
			return err
		}
	}
	return nil
}

func cleanDB(t *testing.T) {
	opt, _ := config.LoadFromEnv("test", time.Second)
	if opt.Storage == "all" {
		opt.Storage = config.StorageFS
	}
	t.Logf("resetting db")
	err := resetDB(opt)
	if err != nil {
		t.Error(err)
	}
	if fedboxApp != nil {
		if st, ok := fedboxApp.Storage.(ls.Resetter); ok {
			st.Reset()
		}
	}
}

func seedTestData(t *testing.T, testData []string) {
	if t == nil {
		panic("invalid test context")
	}
	t.Helper()

	if len(testData) == 0 {
		return
	}

	fields:= logrus.Fields{"action":"seeding", "storage": Options.Storage, "path": Options.StoragePath}
	l := logrus.New()
	db, aDb, err := app.Storage(Options, l.WithFields(fields))
	if err != nil {
		panic(err)
	}
	if err = cmd.Bootstrap(Options); err != nil {
		panic(err)
	}

	o := cmd.New(aDb, db, Options)
	mocks := make(pub.ItemCollection, 0)
	json := loadMockJson("mocks/application.json", nil)()
	act, err := pub.UnmarshalJSON([]byte(json))
	if err == nil {
		mocks = append(mocks, act)
		if clSaver, ok := aDb.(app.ClientSaver); ok {
			clSaver.CreateClient(&osin.DefaultClient{
				Id:          defaultTestApp.Id,
				Secret:      "hahah",
				RedirectUri: "http://127.0.0.1:9998/callback",
				UserData:    nil,
			})
		}
	}

	for _, path := range testData {
		json := loadMockJson(path, nil)()
		if json == "" {
			continue
		}
		it, err := pub.UnmarshalJSON([]byte(json))
		if err == nil && !mocks.Contains(it) {
			mocks = append(mocks, it)
		}
	}
	addMockObjects(o.Storage, mocks, t.Errorf)

	tok, err := o.GenAuthToken(defaultTestApp.Id, defaultTestAccount.Id, nil)
	if err == nil {
		defaultTestAccount.AuthToken = tok
	}
}

var Options config.Options

func SetupAPP(e env.Type) *app.FedBOX {
	Options, _ = config.LoadFromEnv(e, time.Second)
	if Options.Storage == "all" {
		Options.Storage = config.StorageFS
	}
	fields:= logrus.Fields{"action":"running", "storage": Options.Storage, "path": Options.StoragePath}
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)

	r := chi.NewRouter()
	r.Use(log.NewStructuredLogger(l))

	db, o, err := app.Storage(Options, l.WithFields(fields))
	if err != nil {
		panic(err)
	}
	a, _ := app.New(l, "HEAD", Options, db, o)
	return a
}
