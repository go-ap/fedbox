// +build storage_sqlite

package app

import (
	auth "github.com/go-ap/auth/fs"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/storage/sqlite"
	st "github.com/go-ap/storage"
	"github.com/openshift/osin"
	"github.com/sirupsen/logrus"
)

func Storage(c config.Options, l logrus.FieldLogger) (st.Store, osin.Storage, error) {
	l.Debugf("Initializing sqlite storage at %s", c.StoragePath)
	oauth := auth.New(auth.Config{
		Path:  c.BaseStoragePath(),
		LogFn: InfoLogFn(l),
		ErrFn: ErrLogFn(l),
	})
	db, err := sqlite.New(sqlite.Config{
		StoragePath: c.StoragePath,
		Env:         string(c.Env),
		BaseURL:     c.BaseURL,
	})

	if err != nil {
		return nil, nil, errors.Annotatef(err, "unable to connect to sqlite storage")
	}
	return db, oauth, nil
}
