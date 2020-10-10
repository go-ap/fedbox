// +build storage_badger

package cmd

import "github.com/go-ap/fedbox/storage/badger"

var bootstrapFn = badger.Bootstrap

var cleanFn = badger.Clean
