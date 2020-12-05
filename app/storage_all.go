// +build storage_all !storage_pgx,!storage_boltdb,!storage_fs,!storage_badger,!storage_sqlite

package app

import (
	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/storage/badger"
	"github.com/go-ap/fedbox/storage/boltdb"
	"github.com/go-ap/fedbox/storage/fs"
	"github.com/go-ap/fedbox/storage/pgx"
	"github.com/go-ap/fedbox/storage/sqlite"
	st "github.com/go-ap/storage"
	"github.com/openshift/osin"
	"github.com/sirupsen/logrus"
)

func getBadgerStorage(c config.Options, l logrus.FieldLogger) (st.Repository, osin.Storage, error) {
	db := badger.New(badger.Config{
		Path:  c.Badger(),
		LogFn: InfoLogFn(l),
		ErrFn: ErrLogFn(l),
	}, c.BaseURL)
	oauth := auth.NewBoltDBStore(auth.BoltConfig{
		Path:       c.BoltDBOAuth2(),
		BucketName: c.Host,
		LogFn:      InfoLogFn(l),
		ErrFn:      ErrLogFn(l),
	})
	return db, oauth, nil
}

func getBoltStorage(c config.Options, l logrus.FieldLogger) (st.Repository, osin.Storage, error) {
	db := boltdb.New(boltdb.Config{
		Path:  c.BoltDB(),
		LogFn: InfoLogFn(l),
		ErrFn: ErrLogFn(l),
	}, c.BaseURL)

	oauth := auth.NewBoltDBStore(auth.BoltConfig{
		Path:       c.BoltDBOAuth2(),
		BucketName: c.Host,
		LogFn:      InfoLogFn(l),
		ErrFn:      ErrLogFn(l),
	})
	return db, oauth, nil
}

func getFsStorage(c config.Options, l logrus.FieldLogger) (st.Repository, osin.Storage, error) {
	oauth := auth.NewFSStore(auth.FSConfig{
		Path:  c.BaseStoragePath(),
		LogFn: InfoLogFn(l),
		ErrFn: ErrLogFn(l),
	})
	db, err := fs.New(c)
	if err != nil {
		return nil, oauth, err
	}
	return db, oauth, nil
}

func getSqliteStorage(c config.Options, l logrus.FieldLogger) (st.Repository, osin.Storage, error) {
	oauth := auth.NewFSStore(auth.FSConfig{
		Path:  c.BaseStoragePath(),
		LogFn: InfoLogFn(l),
		ErrFn: ErrLogFn(l),
	})
	db, err := sqlite.New(c)
	if err != nil {
		return nil, oauth, err
	}
	return db, oauth, nil
}


func getPgxStorage(c config.Options, l logrus.FieldLogger) (st.Repository, osin.Storage, error) {
	// @todo(marius): we're no longer loading SQL db config env variables
	conf := config.BackendConfig{}
	db, err := pgx.New(conf, c.BaseURL, l)

	oauth := auth.NewPgDBStore(auth.PgConfig{
		Enabled: true,
		Host:    conf.Host,
		Port:    conf.Port,
		User:    conf.User,
		Pw:      conf.Pw,
		Name:    conf.Name,
		LogFn:   InfoLogFn(l),
		ErrFn:   ErrLogFn(l),
	})
	return db, oauth, err
}

func Storage(c config.Options, l logrus.FieldLogger) (st.Repository, osin.Storage, error) {
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
