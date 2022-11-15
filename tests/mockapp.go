//go:build integration

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
	"os"
	"path"
	"strings"
	"testing"
	"text/template"
	"time"

	"git.sr.ht/~mariusor/lw"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/fedbox"
	"github.com/go-ap/fedbox/internal/cmd"
	"github.com/go-ap/fedbox/internal/config"
	ls "github.com/go-ap/fedbox/storage"
	"github.com/go-ap/httpsig"
	"github.com/go-ap/jsonld"
	"github.com/go-ap/processing"
	"github.com/openshift/osin"
	"golang.org/x/crypto/ed25519"
)

func jsonldMarshal(i vocab.Item) string {
	j, err := jsonld.Marshal(i)
	if err != nil {
		panic(err)
	}
	return string(j)
}

func loadMockJson(file string, model interface{}) func() (string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return func() (string, error) { return "", err }
	}
	data = bytes.Trim(data, "\x00")

	t := template.Must(template.New(fmt.Sprintf("mock_%s", path.Base(file))).
		Funcs(template.FuncMap{"json": jsonldMarshal}).Parse(string(data)))

	return func() (string, error) {
		bytes := bytes.Buffer{}
		err := t.Execute(&bytes, model)
		return bytes.String(), err
	}
}

func addMockObjects(r processing.Store, obj vocab.ItemCollection, errFn fedbox.LogFn) error {
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
	if err := cmd.Reset(opt); err != nil {
		t.Error(err)
	}
	if fedboxApp != nil {
		if st, ok := fedboxApp.Storage().(ls.Resetter); ok {
			st.Reset()
		}
	}

	// As we're using ioutil.Tempdir for the storage path, we can remove it fully
	os.RemoveAll(path.Clean(opt.StoragePath))
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

func loadMockFromDisk(file string, model interface{}) vocab.Item {
	json, err := loadMockJson(file, model)()
	if err != nil {
		panic(err)
	}
	it, err := vocab.UnmarshalJSON([]byte(json))
	if err != nil {
		panic(err)
	}
	return it
}

func seedTestData(t *testing.T, testData []string, options config.Options) {
	if t == nil {
		panic("invalid test context")
	}
	t.Helper()

	if len(testData) == 0 {
		return
	}

	fields := lw.Ctx{"action": "seeding", "storage": options.Storage, "path": options.StoragePath}
	l := lw.Dev(lw.SetLevel(lw.DebugLevel)).WithContext(fields)
	db, aDb, err := fedbox.Storage(options, l)
	if err != nil {
		panic(err)
	}
	clientCode := path.Base(defaultTestApp.Id)

	mocks := make(vocab.ItemCollection, 0)
	o := cmd.New(aDb, db, options, l)
	act := loadMockFromDisk("mocks/c2s/actors/application.json", nil)
	mocks = append(mocks, act)
	if clSaver, ok := aDb.(fedbox.ClientSaver); ok {
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
		if !it.GetLink().Contains(vocab.IRI(options.BaseURL), false) {
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
			err = metaSaver.SaveMetadata(ls.Metadata{PrivateKey: pem.EncodeToMemory(&r)}, vocab.IRI(defaultTestAccountC2S.Id))
			if err != nil {
				l.Critf("%s\n", err)
			}
		}
		if tok, err := o.GenAuthToken(clientCode, defaultTestAccountC2S.Id, nil); err == nil {
			defaultTestAccountC2S.AuthToken = tok
		}
	}
}

func SetupAPP(options config.Options) *fedbox.FedBOX {
	if options.Storage == "all" {
		options.Storage = config.StorageFS
	}

	fields := lw.Ctx{"action": "running", "storage": options.Storage, "path": options.BaseStoragePath()}

	l := lw.Dev(lw.SetLevel(options.LogLevel))
	db, o, err := fedbox.Storage(options, l.WithContext(fields))
	if err != nil {
		panic(err)
	}
	if err = cmd.Bootstrap(options); err != nil {
		panic(err)
	}
	a, err := fedbox.New(l, "HEAD", options, db, o)
	if err != nil {
		panic(err)
	}
	if options.Storage == config.StorageFS {
		time.Sleep(100 * time.Millisecond)
	}
	return a
}
