package main

import (
	"os"
	"runtime/debug"

	"github.com/go-ap/fedbox"
)

func main() {
	if build, ok := debug.ReadBuildInfo(); ok && fedbox.AppVersion == "HEAD" {
		if build.Main.Version != "(devel)" {
			fedbox.AppVersion = build.Main.Version
		}
		for _, bs := range build.Settings {
			if bs.Key == "vcs.revision" {
				fedbox.AppVersion = bs.Value[:8]
			}
			if bs.Key == "vcs.modified" {
				fedbox.AppVersion += "-git"
			}
		}
	}
	if err := fedbox.Run(os.Args[1:]...); err != nil {
		fedbox.Errf(os.Stderr, "%+v", err)
		os.Exit(1)
	}
}
