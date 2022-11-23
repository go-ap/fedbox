//go:build storage_boltdb

package cmd

import "github.com/go-ap/storage-boltdb"

var (
	bootstrapFn = func(conf storageConf) error {
		return boltdb.Bootstrap(boltdb.Config{Path: conf.Path, BaseURL: conf.BaseURL}, conf.BaseURL)
	}
	cleanFn = func(conf storageConf) error {
		return boltdb.Clean(boltdb.Config{Path: conf.Path, BaseURL: conf.BaseURL})
	}
)
