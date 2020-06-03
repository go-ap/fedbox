package fs

import (
	"github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/config"
	"os"
)

func Clean(path string) error {
	return os.RemoveAll(path)
}

func Bootstrap(conf config.Options) error {
	r, err := New(conf)
	if err != nil {
		return err
	}
	return r.CreateService(activitypub.Self(activitypub.DefaultServiceIRI(conf.BaseURL)))
}

