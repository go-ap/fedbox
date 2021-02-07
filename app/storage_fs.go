// +build storage_fs

package app

import (
	auth "github.com/go-ap/auth/fs"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/storage/fs"
	st "github.com/go-ap/storage"
	"github.com/openshift/osin"
	"github.com/sirupsen/logrus"
)

func Storage(c config.Options, l logrus.FieldLogger) (st.Store, osin.Storage, error) {
	path := c.BaseStoragePath()
	l.Debugf("Initializing fs storage at %s", path)
	oauth := auth.New(auth.Config{
		Path: path,
		LogFn: InfoLogFn(l),
		ErrFn: ErrLogFn(l),
	})
	db, err := fs.New(fs.Config{
		StoragePath: path,
		BaseURL:     c.BaseURL,
	})
	if err != nil {
		return nil, oauth, err
	}
	return db, oauth, nil
}
