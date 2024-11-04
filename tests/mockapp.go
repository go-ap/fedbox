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
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"git.sr.ht/~mariusor/lw"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/fedbox"
	"github.com/go-ap/fedbox/internal/cmd"
	"github.com/go-ap/fedbox/internal/config"
	ls "github.com/go-ap/fedbox/storage"
	"github.com/go-ap/jsonld"
	"github.com/go-ap/processing"
	"github.com/openshift/osin"
	"golang.org/x/crypto/ed25519"
)

func jsonldMarshal(i vocab.Item) string {
	j, err := jsonld.Marshal(i)
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}
	return string(j)
}

func loadMockJson(file string, model any) func() (string, error) {
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

func addMockObjects(r processing.Store, obj vocab.ItemCollection) error {
	var err error
	for _, it := range obj {
		if it.GetLink() == "" {
			continue
		}
		if it, err = r.Save(it); err != nil {
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
	if t.Failed() {
		return
	}

	tempPath, err := os.Getwd()
	if err != nil {
		t.Logf("Unable to get current path: %s", err)
	}

	tempPath = path.Clean(filepath.Join(tempPath, filepath.Dir(opt.StoragePath)))
	if !strings.Contains(tempPath, ".cache") {
		t.Logf("Unable to clean path: %s", tempPath)
		return
	}
	t.Logf("Removing path: %s", tempPath)

	// As we're using t.TempDir for the storage path, we can remove it fully
	if err := os.RemoveAll(tempPath); err != nil {
		t.Logf("Unable to remove path: %s: %s", tempPath, err)
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

func loadPrivateKeyFromDisk(file string) crypto.PrivateKey {
	data, err := os.ReadFile(file)
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}
	b, _ := pem.Decode(data)
	if b == nil {
		panic("failed decoding pem")
	}
	prvKey, err := x509.ParsePKCS8PrivateKey(b.Bytes)
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}
	return prvKey
}

func loadMockFromDisk(file string, model any) vocab.Item {
	json, err := loadMockJson(file, model)()
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}
	it, err := vocab.UnmarshalJSON([]byte(json))
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}
	return it
}

func saveMocks(testData []string, app *fedbox.FedBOX, l lw.Logger) error {
	baseIRI := vocab.IRI(app.Config().BaseURL)
	db := app.Storage()
	mocks := make(vocab.ItemCollection, 0)
	for _, path := range testData {
		it := loadMockFromDisk(path, nil)
		if !it.GetLink().Contains(baseIRI, false) {
			continue
		}
		if !mocks.Contains(it) {
			mocks = append(mocks, it)
		}
	}
	if err := addMockObjects(db, mocks); err != nil {
		return err
	}

	o, _ := cmd.New(db, app.Config(), l)

	if strings.Contains(defaultTestAccountC2S.Id, app.Config().BaseURL) {
		if err := saveMetadataForActor(defaultTestAccountC2S, db.(ls.MetadataTyper)); err != nil {
			l.Critf("%s\n", err)
		}
		tok, err := o.GenAuthToken(defaultTestApp.Id, defaultTestAccountC2S.Id, nil)
		if err != nil {
			l.Errorf("%+s\n", err)
		} else {
			defaultTestAccountC2S.AuthToken = tok
		}
	}
	if strings.Contains(defaultTestAccountS2S.Id, app.Config().BaseURL) {
		if err := saveMetadataForActor(defaultTestAccountS2S, db.(ls.MetadataTyper)); err != nil {
			l.Critf("%s\n", err)
		}
	}
	return nil
}

func saveMetadataForActor(act testAccount, metaSaver ls.MetadataTyper) error {
	prvEnc, err := x509.MarshalPKCS8PrivateKey(act.PrivateKey)
	if err != nil {
		return err
	}
	r := pem.Block{Type: "PRIVATE KEY", Bytes: prvEnc}
	err = metaSaver.SaveMetadata(
		processing.Metadata{PrivateKey: pem.EncodeToMemory(&r)},
		vocab.IRI(act.Id),
	)
	return nil
}

func seedTestData(app *fedbox.FedBOX) error {
	clientCode := path.Base(defaultTestApp.Id)

	db := app.Storage()

	act := loadMockFromDisk("mocks/c2s/actors/application.json", nil)
	if err := addMockObjects(db, vocab.ItemCollection{act}); err != nil {
		return err
	}

	return db.CreateClient(&osin.DefaultClient{
		Id:          clientCode,
		Secret:      "hahah",
		RedirectUri: "http://127.0.0.1:9998/callback",
		UserData:    nil,
	})
}

func RunTestFedBOX(options config.Options) (*fedbox.FedBOX, error) {
	if options.Storage == "all" {
		options.Storage = config.StorageFS
	}

	fields := lw.Ctx{"action": "running", "storage": options.Storage, "path": options.BaseStoragePath()}

	l := lw.Dev(lw.SetLevel(options.LogLevel))
	db, err := fedbox.Storage(options, l.WithContext(fields))
	if err != nil {
		return nil, err
	}

	a, err := fedbox.New(l, "HEAD", options, db)
	if err != nil {
		return nil, err
	}
	if err := seedTestData(a); err != nil {
		return nil, err
	}

	return a, nil
}
