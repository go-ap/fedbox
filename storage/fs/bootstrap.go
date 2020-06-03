package fs

import (
	"github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/config"
	"os"
	"path"
)

func Clean(conf config.Options) error {
	return os.RemoveAll(path.Join(conf.StoragePath, string(conf.Env)))
}

func Bootstrap(conf config.Options) error {
	r, err := New(conf)
	if err != nil {
		return err
	}
	return r.CreateService(activitypub.Self(activitypub.DefaultServiceIRI(conf.BaseURL)))
}

