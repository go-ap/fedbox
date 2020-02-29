package main

import (
	"fmt"
	"github.com/go-ap/fedbox/cmd"
	"github.com/go-ap/fedbox/internal/log"
	"github.com/go-chi/chi"
	"os"
)

var version = "HEAD"

func main() {
	srv := cmd.NewApp(chi.NewRouter(), log.New(), version)
	err := srv.Run(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s", err)
		os.Exit(1)
	}
}
