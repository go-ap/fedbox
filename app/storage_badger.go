// +build storage_badger

package app

import (
	auth "github.com/go-ap/auth/badger"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/storage/badger"
	st "github.com/go-ap/storage"
	"github.com/openshift/osin"
	"github.com/sirupsen/logrus"
)

func Storage(c config.Options, l logrus.FieldLogger) (st.Store, osin.Storage, error) {
	l.Debugf("Initializing badger storage at %s", c.StoragePath)
	db, err := badger.New(badger.Config{
		Path:    c.StoragePath,
		Env:     string(c.Env),
		BaseURL: c.BaseURL,
		LogFn:   InfoLogFn(l),
		ErrFn:   ErrLogFn(l),
	})
	if err != nil {
		return db, nil, err
	}
	oauth := auth.New(auth.Config{
		Path:  c.BadgerOAuth2(),
		Host:  c.Host,
		LogFn: InfoLogFn(l),
		ErrFn: ErrLogFn(l),
	})
	return db, oauth, nil
}
