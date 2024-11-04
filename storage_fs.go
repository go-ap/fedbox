//go:build storage_fs

package fedbox

import (
	"git.sr.ht/~mariusor/lw"
	"github.com/go-ap/fedbox/internal/config"
	st "github.com/go-ap/fedbox/storage"
	fs "github.com/go-ap/storage-fs"
)

func Storage(c config.Options, l lw.Logger) (st.FullStorage, error) {
	c.Storage = config.DefaultStorage
	p := c.BaseStoragePath()
	l = l.WithContext(lw.Ctx{"path": p})
	l.Debugf("Using fs storage")
	db, err := fs.New(fs.Config{
		Path:        p,
		CacheEnable: c.StorageCache,
		Logger:      l,
		UseIndex:    true,
	})
	if err != nil {
		return nil, err
	}
	return db, nil
}
