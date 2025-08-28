package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/alecthomas/kong"
	"github.com/go-ap/fedbox/internal/cmd"
	"github.com/go-ap/fedbox/internal/env"
)

var version = "HEAD"

const defaultWaitDuration = 5 * time.Second

func main() {
	if build, ok := debug.ReadBuildInfo(); ok && version == "HEAD" && build.Main.Version != "(devel)" && build.Main.Version != "" {
		version = build.Main.Version
	}

	ctx := kong.Parse(
		&cmd.FedBOXRun,
		kong.Name(cmd.AppName),
		kong.Description(fmt.Sprintf("%s instance server version %s", cmd.AppName, version)),
		kong.Vars{
			"version":             version,
			"defaultEnv":          string(env.DEV),
			"defaultWaitDuration": defaultWaitDuration.String(),
			"envTypes":            fmt.Sprintf("%s, %s, %s, %s", env.TEST, env.DEV, env.QA, env.PROD),
		},
		kong.Bind(&cmd.Control{}),
	)
	if err := ctx.Run(); err != nil {
		cmd.Errf(err.Error())
		os.Exit(1)
	}
}
