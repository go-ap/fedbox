package main

import (
	"os"

	"github.com/go-ap/fedbox/internal/cmd"
)

var version = "HEAD"

func main() {
	if err := cmd.NewApp(version).Run(os.Args); err != nil {
		cmd.Errf(err.Error())
		os.Exit(1)
	}
}
