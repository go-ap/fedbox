// +build storage_boltdb

package cmd

import "github.com/go-ap/fedbox/storage/boltdb"

var bootstrapFn = boltdb.Bootstrap

var cleanFn = boltdb.Clean
