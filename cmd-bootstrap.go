package fedbox

import (
	"bytes"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"

	"git.sr.ht/~mariusor/lw"
	"git.sr.ht/~mariusor/mask"
	"git.sr.ht/~mariusor/storage-all"
	vocab "github.com/go-ap/activitypub"
	http "github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/config"
)

type ResetCmd struct{}

func (b ResetCmd) Run(ctl *Base) error {
	if err := ctl.Storage.Open(); err != nil {
		return http.Annotatef(err, "Unable to open FedBOX storage for path %s", ctl.Conf.StoragePath)
	}
	defer ctl.Storage.Close()

	err := reset(ctl.Conf, ctl.Logger)
	if err != nil {
		return err
	}
	keyType := ap.KeyTypeED25519
	if ctl.Conf.MastodonCompatible {
		keyType = ap.KeyTypeRSA
	}
	pair, err := ap.GenerateKeyPair(keyType)
	if err != nil {
		return err
	}
	return bootstrap(ctl, ctl.Service, ctl.Logger, pair, nil)
}

type BootstrapCmd struct {
	KeyType  string `help:"Type of keys to generate: ${keyTypes}" enum:"${keyTypes}" default:"${defaultKeyType}"`
	Password string `hidden:""`
}

func (b BootstrapCmd) Run(ctl *Base) error {
	keyType := ap.KeyType(b.KeyType)
	if CTLRun.Storage.Type != "" {
		ctl.Conf.Storage = CTLRun.Storage.Type
	}

	pw, pair := getPwAndKey(ctl, keyType)
	if b.Password != "" {
		pw = []byte(b.Password)
	}
	if err := ctl.Bootstrap(pw, pair); err != nil {
		ctl.Logger.WithContext(lw.Ctx{"err": err}).Warnf("Unable to bootstrap service actor")
	}
	return nil
}

func BootstrapStorage(conf config.Options, service vocab.Item, l lw.Logger, pair *ap.KeyPair) error {
	ctl := Base{
		in:  os.Stdin,
		out: os.Stdout,
		err: os.Stderr,
	}
	if err := setup(&ctl, conf); err != nil {
		return err
	}
	return bootstrap(&ctl, service, l, pair, nil)
}

func bootstrap(ctl *Base, service vocab.Item, l lw.Logger, pair *ap.KeyPair, pw []byte) error {
	conf := ctl.Conf
	path, err := conf.BaseStoragePath()
	if err != nil {
		return err
	}
	initFns := conf.StorageInitFns(l)
	if err := storage.Bootstrap(initFns...); err != nil {
		return http.Annotatef(err, "Unable to create %s path for storage %s", path, conf.Storage)
	}
	l.Infof("Successfully created %s db for storage %s", path, conf.Storage)

	if ctl.Storage == nil {
		ctl.Storage, err = storage.New(initFns...)
		if err != nil {
			return http.Annotatef(err, "Unable to initialize %s path for storage %s", path, conf.Storage)
		}
	}
	if err = ctl.Storage.Open(); err != nil {
		return http.Annotatef(err, "Unable to open %s path for storage %s", path, conf.Storage)
	}
	defer ctl.Storage.Close()

	if err = CreateService(ctl, service, pair, pw); err != nil {
		return http.Annotatef(err, "Unable to create FedBOX service %s for storage %s", service.GetID(), conf.Storage)
	}
	l.WithContext(lw.Ctx{"storage": conf.Storage, "ID": service.GetID()}).Infof("Successfully created FedBOX service")
	return nil
}

func ResetStorage(conf config.Options, l lw.Logger) error {
	return reset(conf, l)
}

func reset(conf config.Options, l lw.Logger) error {
	if err := storage.Clean(conf.StorageInitFns(l)...); err != nil && !errors.Is(err, os.ErrNotExist) {
		return http.Annotatef(err, "Unable to reset %s db for storage %s", conf.StoragePath, conf.Storage)
	}
	l.Infof("Successfully reset %s db for storage %s", conf.StoragePath, conf.Storage)
	return nil
}

func getPwAndKey(ctl *Base, keyType ap.KeyType) ([]byte, *ap.KeyPair) {
	var pair *ap.KeyPair

	// NOTE(marius): try to find if there's a private key in the storage path
	keyFilePath := filepath.Join(ctl.Conf.StoragePath, ctl.Conf.Hostname+".key")
	if prvKey, err := os.ReadFile(keyFilePath); err == nil {
		pair, err = ap.KeyPairFromPrivateBytes(prvKey)
		if err != nil {
			ctl.Logger.Warnf("Unable to decode found private key")
		} else {
			ctl.Logger.WithContext(lw.Ctx{"typ": pair.Type, "valid": pair.Private != nil}).Tracef("Found private key")
			defer os.RemoveAll(keyFilePath)
		}
	}
	if pair == nil {
		var err error
		// NOTE(marius): we generate the key
		pair, err = ap.GenerateKeyPair(keyType)
		if err != nil {
			ctl.Logger.WithContext(lw.Ctx{"typ": keyType}).Warnf("Unable to generate key pair for service")
		} else {
			ctl.Logger.WithContext(lw.Ctx{"typ": keyType}).Tracef("Generated key pair for service")
		}
	}

	// NOTE(marius): try to find if there's a password set in the storage path
	pwFilePath := filepath.Join(ctl.Conf.StoragePath, ctl.Conf.Hostname+".pw")
	pw, err := os.ReadFile(pwFilePath)
	if err == nil {
		pw = bytes.TrimSpace(pw)
		ctl.Logger.WithContext(lw.Ctx{"pw": mask.B(pw)}).Tracef("Found valid password file")
		defer os.RemoveAll(pwFilePath)
	}
	if pw == nil {
		pw = []byte(rand.Text())
		ctl.Logger.WithContext(lw.Ctx{"pw": mask.B(pw)}).Tracef("Generated random password for service")
	}

	return pw, pair
}
