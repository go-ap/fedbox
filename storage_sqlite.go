//go:build storage_sqlite

package fedbox

import (
	"git.sr.ht/~mariusor/lw"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/internal/config"
	st "github.com/go-ap/fedbox/storage"
	sqlite "github.com/go-ap/storage-sqlite"
)

func Storage(c config.Options, l lw.Logger) (st.FullStorage, error) {
	c.Storage = config.DefaultStorage
	path := c.BaseStoragePath()
	l = l.WithContext(lw.Ctx{"path": path})
	l.Debugf("Using sqlite storage")
	db, err := sqlite.New(sqlite.Config{
		Path:        path,
		CacheEnable: c.StorageCache,
		LogFn:       l.Debugf,
		ErrFn:       l.Warnf,
	})

	if err != nil {
		return nil, errors.Annotatef(err, "unable to connect to sqlite storage")
	}
	return db, nil
}
