//go:build storage_all || (!storage_fs && !storage_boltdb && !storage_badger && !storage_sqlite)

package config

const DefaultStorage = ""
