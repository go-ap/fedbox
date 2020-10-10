// +build integration
// +build storage_fs

package tests

import (
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/storage/fs"
)

var resetDB = func(opt config.Options) error {
	fs.Clean(opt)
	return fs.Bootstrap(opt)
}
