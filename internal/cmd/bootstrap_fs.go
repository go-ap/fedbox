// +build storage_fs

package cmd

import "github.com/go-ap/fedbox/storage/fs"

var (
	bootstrapFn = fs.Bootstrap
	cleanFn     = fs.Clean
)
