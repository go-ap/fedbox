//go:build storage_badger

package fedbox

import (
	"unsafe"

	"git.sr.ht/~mariusor/lw"
	auth "github.com/go-ap/auth/badger"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/storage/badger"
	"github.com/go-ap/processing"
	"github.com/openshift/osin"
)

func Storage(c config.Options, l lw.Logger) (processing.Store, osin.Storage, error) {
	path := c.BaseStoragePath()
	if l != nil {
		l.Debugf("Initializing badger storage at %s", path)
	}
	conf := badger.Config{
		Path:    path,
		BaseURL: c.BaseURL,
		Logger:  l,
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
