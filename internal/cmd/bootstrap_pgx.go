//go:build storage_pgx
// +build storage_pgx

package cmd

import (
	"fmt"

	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/storage/pgx"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	bootstrapFn = func(conf config.Options) error {
		var pgRoot string
		// ask for root pw
		fmt.Printf("%s password: ", pgRoot)
		pgPw, _ := terminal.ReadPassword(0)
		fmt.Println()
		return pgx.Bootstrap(conf, pgRoot, pgPw)
	}
	cleanFn = func(conf config.Options) (err error) {
		var pgRoot string
		// ask for root pw
		fmt.Printf("%s password: ", pgRoot)
		pgPw, _ := terminal.ReadPassword(0)
		fmt.Println()
		return pgx.Clean(conf, pgRoot, pgPw)
	}
)
