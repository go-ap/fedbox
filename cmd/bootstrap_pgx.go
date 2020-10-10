// +build storage_pgx

package cmd

import (
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/storage/pgx"
)

var bootstrapFn = func(conf config.Options) error {
	var pgRoot string
	// ask for root pw
	fmt.Printf("%s password: ", pgRoot)
	pgPw, _ := terminal.ReadPassword(0)
	fmt.Println()
	dir, _ := os.Getwd()
	path := path.Join(dir, "init.sql")
	return pgx.Bootstrap(conf, pgRoot, pgPw, path)
}

var cleanFn = func(conf config.Options) (err error) {
	var pgRoot string
	// ask for root pw
	fmt.Printf("%s password: ", pgRoot)
	pgPw, _ := terminal.ReadPassword(0)
	fmt.Println()
	dir, _ := os.Getwd()
	path := path.Join(dir, "init.sql")
	return pgx.Clean(conf, pgRoot, pgPw, path)
}
