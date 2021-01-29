// +build storage_sqlite

package cmd

import "github.com/go-ap/fedbox/storage/sqlite"

var bootstrapFn = sqlite.Bootstrap

var cleanFn = sqlite.Clean
