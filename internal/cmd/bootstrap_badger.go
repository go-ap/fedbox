// +build storage_badger

package cmd

import (
	"os"

	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/storage/badger"
)

var bootstrapFn = badger.Bootstrap

var cleanFn = func(conf config.Options) error {
	os.RemoveAll(conf.BadgerOAuth2(conf.BaseStoragePath()))
	return badger.Clean(conf)
}
