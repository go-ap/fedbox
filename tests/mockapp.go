// +build integration

package tests

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"testing"
	"text/template"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/fedbox/app"
	"github.com/go-ap/fedbox/internal/cmd"
	"github.com/go-ap/fedbox/internal/config"
	ls "github.com/go-ap/fedbox/storage"
	"github.com/go-ap/httpsig"
	"github.com/go-ap/jsonld"
	"github.com/go-ap/storage"
	"github.com/openshift/osin"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ed25519"
)

func jsonldMarshal(i pub.Item) string {
	j, err := jsonld.Marshal(i)
	if err != nil {
		panic(err)
	}
	return string(j)
}

func loadMockJson(file string, model interface{}) func() string {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return func() string { return "" }
	}
	data = bytes.Trim(data, "\x00")

	t := template.Must(template.New(fmt.Sprintf("mock_%s", path.Base(file))).
		Funcs(template.FuncMap{"json": jsonldMarshal}).Parse(string(data)))

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

func cleanDB(t *testing.T, opt config.Options) {
	if opt.Storage == "all" {
		opt.Storage = config.StorageFS
	}
	t.Logf("resetting %q db: %s", opt.Storage, opt.StoragePath)
	err := cmd.Reset(opt)
	if err != nil {
		t.Error(err)
	}
	if fedboxApp != nil {
		if st, ok := fedboxApp.Storage().(ls.Resetter); ok {
			st.Reset()
		}
	}
}

func publicKeyFrom(key crypto.PrivateKey) crypto.PublicKey {
	switch k := key.(type) {
	case *rsa.PrivateKey:
		return k.PublicKey
	case *ecdsa.PrivateKey:
		return k.PublicKey
	case ed25519.PrivateKey:
		return k.Public()
	}
	panic(fmt.Sprintf("Unknown private key type[%T] %v", key, key))
	return nil
}

func keyType(key crypto.PrivateKey) httpsig.Algorithm {
	switch key.(type) {
	case *rsa.PrivateKey:
		return httpsig.RSASHA256
	case ed25519.PrivateKey:
		return httpsig.Ed25519
	}
	panic(fmt.Sprintf("Unknown private key type[%T] %v", key, key))
	return nil
}

func loadPrivateKeyFromDisk(file string) crypto.PrivateKey {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}
	b, _ := pem.Decode(data)
	if b == nil {
		panic("failed decoding pem")
	}
	prvKey, err := x509.ParsePKCS8PrivateKey(b.Bytes)
	if err != nil {
		panic(err)
	}
	return prvKey
}

func loadMockFromDisk(file string, model interface{}) pub.Item {
	json := loadMockJson(file, model)()
	act, err := pub.UnmarshalJSON([]byte(json))
	if err != nil {
		panic(err)
	}
	return act
}

func seedTestData(t *testing.T, testData []string, options config.Options) {
	if t == nil {
		panic("invalid test context")
	}
	t.Helper()

	if len(testData) == 0 {
		return
	}

	fields := logrus.Fields{"action": "seeding", "storage": options.Storage, "path": options.StoragePath}
	l := logger().WithFields(fields)
	db, aDb, err := app.Storage(options, l)
	if err != nil {
		panic(err)
	}
	clientCode := path.Base(defaultTestApp.Id)

	mocks := make(pub.ItemCollection, 0)
	o := cmd.New(aDb, db, options)
	act := loadMockFromDisk("mocks/application.json", nil)
	mocks = append(mocks, act)
	if clSaver, ok := aDb.(app.ClientSaver); ok {
		clSaver.CreateClient(&osin.DefaultClient{
			Id:          clientCode,
			Secret:      "hahah",
			RedirectUri: "http://127.0.0.1:9998/callback",
			UserData:    nil,
		})
	}

	seedMetadataForTestUser := false
	for _, path := range testData {
		it := loadMockFromDisk(path, nil)
		if !it.GetLink().Contains(pub.IRI(options.BaseURL), false) {
			continue
		}
		if it.GetLink().String() == defaultTestAccountC2S.Id {
			seedMetadataForTestUser = true
		}
		if !mocks.Contains(it) {
			mocks = append(mocks, it)
		}
	}
	addMockObjects(db, mocks, t.Errorf)

	if strings.Contains(defaultTestAccountC2S.Id, options.BaseURL) {
		if metaSaver, ok := db.(ls.MetadataTyper); seedMetadataForTestUser && ok {
			l.Infof("Seeding metadata for test user: %s", defaultTestAccountC2S.Id)
			prvEnc, err := x509.MarshalPKCS8PrivateKey(defaultTestAccountC2S.PrivateKey)
			if err != nil {
				panic(err)
			}
			r := pem.Block{Type: "PRIVATE KEY", Bytes: prvEnc}
			err = metaSaver.SaveMetadata(ls.Metadata{PrivateKey: pem.EncodeToMemory(&r)}, pub.IRI(defaultTestAccountC2S.Id))
			if err != nil {
				l.Panicf("%s\n", err.Error())
			}
		}
		if tok, err := o.GenAuthToken(clientCode, defaultTestAccountC2S.Id, nil); err == nil {
			defaultTestAccountC2S.AuthToken = tok
		}
	}
}

func SetupAPP(options config.Options) *app.FedBOX {
	if options.Storage == "all" {
		options.Storage = config.StorageFS
	}
	fields := logrus.Fields{"action": "running", "storage": options.Storage, "path": options.BaseStoragePath()}

	l := logger()
	db, o, err := app.Storage(options, l.WithFields(fields))
	if err != nil {
		panic(err)
	}
	if err = cmd.Bootstrap(options); err != nil {
		panic(err)
	}
	a, _ := app.New(l, "HEAD", options, db, o)
	if options.Storage == config.StorageFS {
		time.Sleep(100 * time.Millisecond)
	}
	return a
}
