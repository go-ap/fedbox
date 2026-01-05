package main

import (
	"os"

	"github.com/go-ap/fedbox/internal/cmd"
)

func main() {
	if err := cmd.Run(); err != nil {
		cmd.Errf("Error: %s", err.Error())
		os.Exit(1)
	}
}
