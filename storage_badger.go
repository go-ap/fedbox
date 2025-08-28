//go:build storage_badger

package fedbox

import (
	"git.sr.ht/~mariusor/lw"
	"github.com/go-ap/fedbox/internal/config"
	st "github.com/go-ap/fedbox/storage"
	"github.com/go-ap/storage-badger"
)

func Storage(c config.Options, l lw.Logger) (st.FullStorage, error) {
	c.Storage = config.DefaultStorage
	path, err := c.BaseStoragePath()
	if err != nil {
		return nil, err
	}
	l = l.WithContext(lw.Ctx{"path": path})
	l.Debugf("Using badger storage")
	conf := badger.Config{
		Path:  path,
		LogFn: l.Debugf,
		ErrFn: l.Warnf,
	}
	db, err := badger.New(conf)
	if err != nil {
		return db, err
	}
	return db, nil
}
