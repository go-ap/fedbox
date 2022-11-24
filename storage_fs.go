//go:build storage_fs

package fedbox

import (
	"git.sr.ht/~mariusor/lw"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/processing"
	fs "github.com/go-ap/storage-fs"
	"github.com/openshift/osin"
)

func Storage(c config.Options, l lw.Logger) (processing.Store, osin.Storage, error) {
	p := c.BaseStoragePath()
	l = l.WithContext(lw.Ctx{"path": p})
	l.Debugf("Initializing fs storage")
	db, err := fs.New(fs.Config{
		Path: p,
		CacheEnable: c.StorageCache,
		LogFn: l.Debugf,
		ErrFn: l.Warnf,
	})
	if err != nil {
		return nil, nil, err
	}
	return db, db, nil
}
