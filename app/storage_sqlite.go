// +build storage_sqlite

package app

import (
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/storage/sqlite"
	st "github.com/go-ap/storage"
	"github.com/openshift/osin"
	"github.com/sirupsen/logrus"
)

func Storage(c config.Options, l logrus.FieldLogger) (st.Store, osin.Storage, error) {
	l.Debugf("Initializing sqlite storage at %s", c.StoragePath)
	db, err := sqlite.New(sqlite.Config{
		StoragePath: c.BaseStoragePath(),
		Env:         string(c.Env),
		BaseURL:     c.BaseURL,
	})
	if err != nil {
		return nil, nil, errors.Annotatef(err, "unable to connect to sqlite storage")
	}
	return db, nil, errors.NotImplementedf("sqlite storage not implemented yet")
}
