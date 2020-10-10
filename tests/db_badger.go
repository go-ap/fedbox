// +build integration
// +build storage_badger

package tests

import (
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/storage/badger"
)

var resetDB = func(opt config.Options) error {
	badger.Clean(opt)
	return badger.Bootstrap(opt)
}
