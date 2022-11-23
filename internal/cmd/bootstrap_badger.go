//go:build storage_badger
// +build storage_badger

package cmd

import (
	"github.com/go-ap/storage-badger"
)

var (
	bootstrapFn = func(conf storageConf) error {
		return badger.Bootstrap(badger.Config{Path: conf.Path}, conf.BaseURL)
	}
	cleanFn = func(conf storageConf) error {
		return badger.Clean(badger.Config{Path: conf.Path})
	}
)
