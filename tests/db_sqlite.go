// +build integration,storage_sqlite

package tests

import (
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/storage/sqlite"
)

var resetDB = func(opt config.Options) error {
	sqlite.Clean(opt)
	return sqlite.Bootstrap(opt)
}
