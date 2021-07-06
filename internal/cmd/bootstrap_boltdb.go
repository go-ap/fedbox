// +build storage_boltdb

package cmd

import "github.com/go-ap/fedbox/storage/boltdb"

var (
	bootstrapFn = boltdb.Bootstrap
	cleanFn     = boltdb.Clean
)
