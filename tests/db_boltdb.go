// +build integration
// +build storage_boltdb

package tests

import (
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/storage/boltdb"
)

var resetDB = func(opt config.Options) error {
	boltdb.Clean(opt)
	return boltdb.Bootstrap(opt)
}
