//go:build storage_fs

package cmd

import (
	fs "github.com/go-ap/storage-fs"
)

var (
	bootstrapFn = func(conf storageConf) error {
		return fs.Bootstrap(fs.Config{Path: conf.Path, CacheEnable: conf.CacheEnable}, conf.BaseURL)
	}
	cleanFn     = func (conf storageConf) error {
		return fs.Clean(fs.Config{Path: conf.Path, CacheEnable: conf.CacheEnable})
	}
)
