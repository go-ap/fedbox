package main

import (
	"os"
	"runtime/debug"

	"github.com/go-ap/fedbox/internal/cmd"
)

var version = "HEAD"

func main() {
	if build, ok := debug.ReadBuildInfo(); ok && version == "HEAD" && build.Main.Version != "(devel)" {
		version = build.Main.Version
	}
	if err := cmd.NewApp(version).Run(os.Args); err != nil {
		cmd.Errf(err.Error())
		os.Exit(1)
	}
}
