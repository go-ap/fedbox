package main

import (
	"github.com/go-ap/fedbox/cmd"
	"github.com/go-chi/chi"
	"os"
)

var version = "HEAD"

func main() {
	if err := cmd.NewApp(chi.NewRouter(), version).Run(os.Args); err != nil {
		os.Exit(1)
	}
}
