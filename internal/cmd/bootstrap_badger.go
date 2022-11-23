//go:build storage_badger
// +build storage_badger

package cmd

import (
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/storage/badger"
)

var (
	bootstrapFn = func(conf storageConf) error {
		return badger.Bootstrap(config.Options{StoragePath: conf.Path, BaseURL: conf.BaseURL})
	}
	cleanFn = func(conf storageConf) error {
		return badger.Clean(config.Options{StoragePath: conf.Path, BaseURL: conf.BaseURL})
	}
)
