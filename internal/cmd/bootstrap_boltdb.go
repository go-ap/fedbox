//go:build storage_boltdb

package cmd

import (
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/storage-boltdb"
)

var (
	bootstrapFn = func(conf storageConf, service vocab.Item) error {
		return boltdb.Bootstrap(boltdb.Config{Path: conf.Path}, service)
	}
	cleanFn = func(conf storageConf) error {
		return boltdb.Clean(boltdb.Config{Path: conf.Path})
	}
)
