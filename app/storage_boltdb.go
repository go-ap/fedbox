// +build storage_boltdb

package app

import (
	"github.com/go-ap/auth"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/storage/boltdb"
	st "github.com/go-ap/storage"
	"github.com/openshift/osin"
	"github.com/sirupsen/logrus"
)

func getBoltStorage(c config.Options, l logrus.FieldLogger) (st.Repository, osin.Storage, error) {
	db := boltdb.New(boltdb.Config{
		Path:  c.BoltDB(),
		LogFn: InfoLogFn(l),
		ErrFn: ErrLogFn(l),
	}, c.BaseURL)

	oauth := auth.NewBoltDBStore(auth.BoltConfig{
		Path:       c.BoltDBOAuth2(),
		BucketName: c.Host,
		LogFn:      InfoLogFn(l),
		ErrFn:      ErrLogFn(l),
	})
	return db, oauth, nil
}

func Storage(c config.Options, l logrus.FieldLogger) (st.Repository, osin.Storage, error) {
	return getBoltStorage(c, l)
}
