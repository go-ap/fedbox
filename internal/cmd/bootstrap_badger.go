//go:build storage_badger
// +build storage_badger

package cmd

import (
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/storage-badger"
)

var (
	bootstrapFn = func(conf storageConf, service vocab.Item) error {
		return badger.Bootstrap(badger.Config{Path: conf.Path}, service)
	}
	cleanFn = func(conf storageConf) error {
		return badger.Clean(badger.Config{Path: conf.Path})
	}
)
