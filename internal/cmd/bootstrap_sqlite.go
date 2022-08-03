//go:build storage_sqlite
// +build storage_sqlite

package cmd

import (
	auth "github.com/go-ap/auth/sqlite"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/storage/sqlite"
)

var (
	bootstrapFn = func(conf config.Options) error {
		if err := auth.Bootstrap(auth.Config{Path: conf.BaseStoragePath()}, nil); err != nil {
			return err
		}
		return sqlite.Bootstrap(conf)
	}
	cleanFn = func(conf config.Options) error {
		return sqlite.Clean(conf)
	}
)
