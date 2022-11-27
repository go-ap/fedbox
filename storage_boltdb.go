//go:build storage_boltdb

package fedbox

import (
	"git.sr.ht/~mariusor/lw"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/processing"
	"github.com/go-ap/storage-boltdb"
	"github.com/openshift/osin"
)

func Storage(c config.Options, l lw.Logger) (processing.Store, osin.Storage, error) {
	path := c.BaseStoragePath()
	l = l.WithContext(lw.Ctx{"path": path})
	l.Debugf("Initializing boltdb storage")
	db, err := boltdb.New(boltdb.Config{
		Path:  path,
		LogFn: l.Debugf,
		ErrFn: l.Warnf,
	})
	if err != nil {
		return nil, nil, err
	}
	return db, db, nil
}
