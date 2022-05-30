//go:build storage_fs
// +build storage_fs

package app

import (
	"path"

	auth "github.com/go-ap/auth/fs"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/storage/fs"
	"github.com/go-ap/processing"
	"github.com/openshift/osin"
	"github.com/sirupsen/logrus"
)

func Storage(c config.Options, l logrus.FieldLogger) (processing.Store, osin.Storage, error) {
	p := c.BaseStoragePath()
	l.Debugf("Initializing fs storage at %s", p)
	oauth := auth.New(auth.Config{
		Path:  p,
		LogFn: InfoLogFn(l),
		ErrFn: ErrLogFn(l),
	})
	db, err := fs.New(fs.Config{
		StoragePath: path.Dir(p),
		BaseURL:     c.BaseURL,
		EnableCache: c.StorageCache,
	})
	if err != nil {
		return nil, oauth, err
	}
	return db, oauth, nil
}
