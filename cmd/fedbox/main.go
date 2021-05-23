package main

import (
	"github.com/go-ap/fedbox/internal/cmd"
	"os"
)

var version = "HEAD"

func main() {
	if err := cmd.NewApp(version).Run(os.Args); err != nil {
		cmd.Errf(err.Error())
		os.Exit(1)
	}
}
