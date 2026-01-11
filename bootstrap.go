package fedbox

import (
	"errors"
	"os"

	"git.sr.ht/~mariusor/lw"
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
	return bootstrap(ctl.Conf, ctl.Service, ctl.Logger)
}

type BootstrapCmd struct {
	KeyType string `help:"Type of keys to generate: ${keyTypes}" enum:"${keyTypes}" default:"${defaultKeyType}"`
}

func (b BootstrapCmd) Run(ctl *Base) error {
	keyType := b.KeyType
	ctl.Conf.Storage = CTLRun.Storage.Type
	ctl.Service = ap.Self(ap.DefaultServiceIRI(ctl.Conf.BaseURL))
	if err := bootstrap(ctl.Conf, ctl.Service, ctl.Logger); err != nil {
		Errf(ctl.err, "Error adding service: %s\n", err)
		return err
	}
	if metaSaver, ok := ctl.Storage.(ap.MetadataStorage); ok {
		if err := ap.AddKeyToItem(metaSaver, &ctl.Service, keyType); err != nil {
			Errf(ctl.err, "Error saving metadata for service: %s", err)
			return err
		}
	}
	return nil
}

func BootstrapStorage(conf config.Options, service vocab.Item, l lw.Logger) error {
	return bootstrap(conf, service, l)
}

func bootstrap(conf config.Options, service vocab.Item, l lw.Logger) error {
	path, err := conf.BaseStoragePath()
	if err != nil {
		return err
	}
	initFns := conf.StorageInitFns(l)
	if err := storage.Bootstrap(initFns...); err != nil {
		return http.Annotatef(err, "Unable to create %s path for storage %s", path, conf.Storage)
	}
	l.Infof("Successfully created %s db for storage %s", path, conf.Storage)

	st, err := storage.New(initFns...)
	if err != nil {
		return http.Annotatef(err, "Unable to initialize %s path for storage %s", path, conf.Storage)
	}
	if err = st.Open(); err != nil {
		return http.Annotatef(err, "Unable to open %s path for storage %s", path, conf.Storage)
	}
	defer st.Close()

	keyType := ap.KeyTypeRSA
	if conf.MastodonIncompatible {
		keyType = ap.KeyTypeED25519
	}
	if err = ap.CreateService(st, service, keyType); err != nil {
		return http.Annotatef(err, "Unable to create FedBOX service %s for storage %s", service.GetID(), conf.Storage)
	}
	l.Infof("Successfully created FedBOX service %s for storage %s", service.GetID(), conf.Storage)
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
