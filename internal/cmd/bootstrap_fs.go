//go:build storage_fs

package cmd

import (
	vocab "github.com/go-ap/activitypub"
	fs "github.com/go-ap/storage-fs"
)

var (
	bootstrapFn = func(conf storageConf, service vocab.Item) error {
		return fs.Bootstrap(fs.Config{Path: conf.Path, CacheEnable: conf.CacheEnable}, service)
	}
	cleanFn = func(conf storageConf) error {
		return fs.Clean(fs.Config{Path: conf.Path, CacheEnable: conf.CacheEnable})
	}
)
