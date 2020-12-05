// +build storage_sqlite

package app

import (
	"github.com/go-ap/auth"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/storage/sqlite"
	st "github.com/go-ap/storage"
	"github.com/openshift/osin"
	"github.com/sirupsen/logrus"
)

func Storage(c config.Options, l logrus.FieldLogger) (st.Repository, osin.Storage, error) {
	oauth := auth.NewSqliteStore(auth.SqliteConfig{
		Path:  c.BaseStoragePath(),
		LogFn: InfoLogFn(l),
		ErrFn: ErrLogFn(l),
	})
	db, err := sqlite.New(c)
	if err != nil {
		return nil, oauth, err
	}
	return db, oauth, err
}
