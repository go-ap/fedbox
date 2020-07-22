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
	"github.com/go-ap/fedbox/storage/badger"
	"github.com/go-ap/fedbox/storage/boltdb"
	"github.com/go-ap/fedbox/storage/fs"
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

func seedTestData(t *testing.T, testData []string, reset bool) {
	if t == nil {
		panic("invalid test context")
	}
	t.Helper()
	if reset {
		resetDB()
	}

	opt, _ := config.LoadFromEnv("test")
	u, _ := url.Parse(apiURL)

	b, err := getStorage(opt, u)
	if err != nil {
		t.Errorf("Unable to initialize storage for tests: %s", err)
		return
	}
	if b == nil {
		t.Errorf("Unable to initialize storage for tests: nil returned")
		return
	}
	s := getOAuthStorage(opt, u)

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

func resetDB() string {
	opt, _ := config.LoadFromEnv("test")
	var dbPath string
	if opt.Storage == config.StorageBoltDB {
		boltdb.Clean(opt)
		boltdb.Bootstrap(opt)
	}
	if opt.Storage == config.StorageBadger {
		dbPath, _ = badger.Path(opt)
		badger.Clean(opt)
		badger.Bootstrap(opt)
	}
	if opt.Storage == config.StorageFS {
		fs.Clean(opt)
		fs.Bootstrap(opt)
	}
	return dbPath
}

func getBadgerStorage(opt config.Options, u *url.URL) storage.Repository {
	path, _ := badger.Path(config.Options{
		StoragePath: opt.StoragePath,
		Env:         opt.Env,
		Host:        u.Host,
	})
	return badger.New(badger.Config{
		Path: path,
	}, u.String())
}

func getFsStorage(opt config.Options, u *url.URL) (storage.Repository, error) {
	return fs.New(opt)
}

func getBoltDBStorage(opt config.Options, u *url.URL) storage.Repository {
	path := config.GetDBPath(opt.StoragePath, u.Host, opt.Env)
	l := logrus.New()
	return boltdb.New(boltdb.Config{
		Path:  path,
		LogFn: app.InfoLogFn(l),
		ErrFn: app.ErrLogFn(l),
	}, u.String())
}

func getOAuthStorage(opt config.Options, u *url.URL) osin.Storage {
	l := logrus.New()
	pathOauth := config.GetDBPath(opt.StoragePath, fmt.Sprintf("%s-oauth", u.Host), opt.Env)
	if _, err := os.Stat(pathOauth); os.IsNotExist(err) {
		err := auth.BootstrapBoltDB(pathOauth, []byte(host))
		if err != nil {
			l.Errorf("Unable to create missing boltdb file %s: %s", pathOauth, err)
		}
	}
	return auth.NewBoltDBStore(auth.BoltConfig{
		Path:       pathOauth,
		BucketName: u.Host,
		LogFn:      app.InfoLogFn(l),
		ErrFn:      app.ErrLogFn(l),
	})
}

func getStorage(opt config.Options, u *url.URL) (storage.Repository, error) {
	if opt.Storage == config.StorageBoltDB {
		return getBoltDBStorage(opt, u), nil
	}
	if opt.Storage == config.StorageBadger {
		return getBadgerStorage(opt, u), nil
	}
	if opt.Storage == config.StorageFS {
		return getFsStorage(opt, u)
	}
	return nil, nil
}

func runAPP(e env.Type) int {
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)

	opt, _ := config.LoadFromEnv(env.TEST)
	u, _ := url.Parse(apiURL)

	r := chi.NewRouter()
	r.Use(log.NewStructuredLogger(l))

	o := osinServer(getOAuthStorage(opt, u), l)

	a, _ := app.New(l, "HEAD", string(e))
	a.Storage, _ = getStorage(opt, u)

	r.Use(app.Repo(a.Storage))
	r.Route("/", a.Routes(a.Config().BaseURL, o, l))

	return a.Run(r, time.Second)
}
