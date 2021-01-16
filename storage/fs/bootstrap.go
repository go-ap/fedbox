// +build storage_fs storage_all !storage_boltdb,!storage_badger,!storage_pgx

package fs

import (
	"github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/cache"
	"github.com/go-ap/fedbox/internal/config"
	"os"
	"path"
)

func Clean(conf config.Options) error {
	return os.RemoveAll(path.Join(conf.StoragePath, string(conf.Env)))
}

func Bootstrap(conf config.Options) error {
	r, err := New(Config{
		StoragePath: conf.StoragePath,
		Env:         string(conf.Env),
		BaseURL:     conf.BaseURL,
	})
	if err != nil {
		return err
	}
	return r.CreateService(activitypub.Self(activitypub.DefaultServiceIRI(conf.BaseURL)))
}

func (r *repo) Reset() {
	r.cache = cache.New(true)
}

