//go:build storage_fs

package fedbox

import (
	"git.sr.ht/~mariusor/lw"
	"github.com/go-ap/fedbox/internal/config"
	fs "github.com/go-ap/storage-fs"
)

func Storage(c config.Options, l lw.Logger) (FullStorage, error) {
	c.Storage = config.DefaultStorage
	p := c.BaseStoragePath()
	l = l.WithContext(lw.Ctx{"path": p})
	l.Debugf("Using fs storage")
	db, err := fs.New(fs.Config{
		Path:        p,
		CacheEnable: c.StorageCache,
		LogFn:       l.Debugf,
		ErrFn:       l.Warnf,
	})
	if err != nil {
		return nil, err
	}
	return db, nil
}
