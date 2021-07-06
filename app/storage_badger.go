// +build storage_badger

package app

import (
	"unsafe"

	auth "github.com/go-ap/auth/badger"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/storage/badger"
	st "github.com/go-ap/storage"
	"github.com/openshift/osin"
	"github.com/sirupsen/logrus"
)

func Storage(c config.Options, l logrus.FieldLogger) (st.Store, osin.Storage, error) {
	path := c.BaseStoragePath()
	conf := badger.Config{
		Path:    path,
		BaseURL: c.BaseURL,
	}
	if l != nil {
		l.Debugf("Initializing badger storage at %s", path)
		conf.LogFn = InfoLogFn(l)
		conf.ErrFn = ErrLogFn(l)
	}
	db, err := badger.New(conf)
	if err != nil {
		return db, nil, err
	}
	authConf := (*auth.Config)(unsafe.Pointer(&conf))
	authConf.Path = c.BadgerOAuth2(path)
	oauth := auth.New(*authConf)
	return db, oauth, nil
}
