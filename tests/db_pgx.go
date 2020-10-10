// +build integration
// +build storage_pgx

package tests

import (
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/storage/pgx"
)

var resetDB = func(opt config.Options) error {
	pgx.Clean(opt)
	return pgx.Bootstrap(opt)
}
