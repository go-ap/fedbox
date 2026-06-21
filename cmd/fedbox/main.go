package main

import (
	"os"
	"runtime/debug"

	"github.com/go-ap/fedbox"
)

func main() {
	if build, ok := debug.ReadBuildInfo(); ok && fedbox.AppVersion == "HEAD" && build.Main.Version != "(devel)" {
		fedbox.AppVersion = build.Main.Version
	}
	if err := fedbox.Run(os.Args[1:]...); err != nil {
		fedbox.Errf(os.Stderr, "%+v", err)
		os.Exit(1)
	}
}
