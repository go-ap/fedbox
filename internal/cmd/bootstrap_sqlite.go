//go:build storage_sqlite

package cmd

import (
	auth "github.com/go-ap/auth/sqlite"
	sqlite "github.com/go-ap/storage-sqlite"
	"path/filepath"
)

func conf(opt storageConf) sqlite.Config {
	return sqlite.Config{
		Path:        opt.Path,
		CacheEnable: opt.CacheEnable,
	}
}

var (
	bootstrapFn = func(opt storageConf) error {
		abs, err := filepath.Abs(opt.Path)
		if err != nil {
			return err
		}
		opt.Path = abs
		if err := auth.Bootstrap(auth.Config{Path: opt.Path}, nil); err != nil {
			return err
		}
		return sqlite.Bootstrap(conf(opt), opt.BaseURL)
	}
	cleanFn = func(opt storageConf) error {
		abs, err := filepath.Abs(opt.Path)
		if err != nil {
			return err
		}
		opt.Path = abs
		return sqlite.Clean(conf(opt))
	}
)
