// +build storage_boltdb

package app

import (
	auth "github.com/go-ap/auth/boltdb"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/storage/boltdb"
	st "github.com/go-ap/storage"
	"github.com/openshift/osin"
	"github.com/sirupsen/logrus"
)

func Storage(c config.Options, l logrus.FieldLogger) (st.Store, osin.Storage, error) {
	l.Debugf("Initializing boltdb storage at %s", c.BoltDB())
	db := boltdb.New(boltdb.Config{
		Path:  c.BoltDB(),
		LogFn: InfoLogFn(l),
		ErrFn: ErrLogFn(l),
	}, c.BaseURL)

	oauth := auth.New(auth.Config{
		Path:       c.BoltDBOAuth2(),
		BucketName: c.Host,
		LogFn:      InfoLogFn(l),
		ErrFn:      ErrLogFn(l),
	})
	return db, oauth, nil
}
