//go:build storage_all || (!storage_pgx && !storage_boltdb && !storage_fs && !storage_badger && !storage_sqlite)

package fedbox

import (
	"path"
	"unsafe"

	"git.sr.ht/~mariusor/lw"
	authbadger "github.com/go-ap/auth/badger"
	authboltdb "github.com/go-ap/auth/boltdb"
	authfs "github.com/go-ap/auth/fs"
	authpgx "github.com/go-ap/auth/pgx"
	authsqlite "github.com/go-ap/auth/sqlite"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/storage/badger"
	"github.com/go-ap/fedbox/storage/boltdb"
	"github.com/go-ap/fedbox/storage/fs"
	"github.com/go-ap/fedbox/storage/pgx"
	"github.com/go-ap/fedbox/storage/sqlite"
	"github.com/go-ap/processing"
	"github.com/openshift/osin"
)

func getBadgerStorage(c config.Options, l lw.Logger) (processing.Store, osin.Storage, error) {
	path := c.BaseStoragePath()
	conf := badger.Config{
		Path:    path,
		BaseURL: c.BaseURL,
		Logger:  l,
	}
	if l != nil {
		l.Debugf("Initializing badger storage at %s", path)
	}
	db, err := badger.New(conf)
	if err != nil {
		return db, nil, err
	}
	authConf := (*authbadger.Config)(unsafe.Pointer(&conf))
	authConf.Path = c.BadgerOAuth2(path)
	oauth := authbadger.New(*authConf)
	return db, oauth, nil
}

func getBoltStorage(c config.Options, l lw.Logger) (processing.Store, osin.Storage, error) {
	path := c.BaseStoragePath()
	l.Debugf("Initializing boltdb storage at %s", path)
	db, err := boltdb.New(boltdb.Config{
		Path:    path,
		BaseURL: c.BaseURL,
		LogFn:   InfoLogFn(l),
		ErrFn:   ErrLogFn(l),
	})
	if err != nil {
		return nil, nil, err
	}

	oauth := authboltdb.New(authboltdb.Config{
		Path:       c.BoltDBOAuth2(),
		BucketName: c.Host,
		LogFn:      InfoLogFn(l),
		ErrFn:      ErrLogFn(l),
	})
	return db, oauth, nil
}

func getFsStorage(c config.Options, l lw.Logger) (processing.Store, osin.Storage, error) {
	p := c.BaseStoragePath()
	l.Debugf("Initializing fs storage at %s", c.BaseStoragePath())
	oauth := authfs.New(authfs.Config{
		Path:  p,
		LogFn: InfoLogFn(l),
		ErrFn: ErrLogFn(l),
	})
	db, err := fs.New(fs.Config{
		StoragePath: path.Dir(p),
		BaseURL:     c.BaseURL,
		EnableCache: c.StorageCache,
	})
	if err != nil {
		return nil, oauth, err
	}
	return db, oauth, nil
}

func getSqliteStorage(c config.Options, l lw.Logger) (processing.Store, osin.Storage, error) {
	path := c.BaseStoragePath()
	l.Debugf("Initializing sqlite storage at %s", path)
	oauth := authsqlite.New(authsqlite.Config{
		Path:  path,
		LogFn: InfoLogFn(l),
		ErrFn: ErrLogFn(l),
	})
	db, err := sqlite.New(sqlite.Config{
		StoragePath: path,
		BaseURL:     c.BaseURL,
		EnableCache: c.StorageCache,
	})
	if err != nil {
		return nil, nil, errors.Annotatef(err, "unable to connect to sqlite storage")
	}
	return db, oauth, nil
}

func getPgxStorage(c config.Options, l lw.Logger) (processing.Store, osin.Storage, error) {
	// @todo(marius): we're no longer loading SQL db config env variables
	l.Debugf("Initializing pgx storage at %s", c.StoragePath)
	conf := pgx.Config{}
	db, err := pgx.New(conf, c.BaseURL, l)
	if err != nil {
		return nil, nil, errors.Annotatef(err, "unable to connect to pgx storage")
	}
	oauth := authpgx.New(authpgx.Config{
		Enabled: true,
		Host:    conf.Host,
		Port:    int64(conf.Port),
		User:    conf.User,
		Pw:      conf.Password,
		Name:    conf.Database,
		LogFn:   InfoLogFn(l),
		ErrFn:   ErrLogFn(l),
	})
	return db, oauth, errors.NotImplementedf("pgx storage is not implemented yet")
}

func Storage(c config.Options, l lw.Logger) (processing.Store, osin.Storage, error) {
	switch c.Storage {
	case config.StorageBoltDB:
		return getBoltStorage(c, l)
	case config.StorageBadger:
		return getBadgerStorage(c, l)
	case config.StoragePostgres:
		return getPgxStorage(c, l)
	case config.StorageSqlite:
		return getSqliteStorage(c, l)
	case config.StorageFS:
		return getFsStorage(c, l)
	}
	return nil, nil, errors.NotImplementedf("Invalid storage type %s", c.Storage)
}
