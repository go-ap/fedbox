//go:build storage_fs

package fedbox

import (
	"git.sr.ht/~mariusor/lw"
	auth "github.com/go-ap/auth/fs"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/processing"
	fs "github.com/go-ap/storage-fs"
	"github.com/openshift/osin"
)

func Storage(c config.Options, l lw.Logger) (processing.Store, osin.Storage, error) {
	p := c.BaseStoragePath()
	l.Debugf("Initializing fs storage at %s", p)
	oauth := auth.New(auth.Config{
		Path:  p,
		LogFn: InfoLogFn(l),
		ErrFn: ErrLogFn(l),
	})
	db, err := fs.New(fs.Config{Path: p, CacheEnable: c.StorageCache})
	if err != nil {
		return nil, oauth, err
	}
	return db, oauth, nil
}
