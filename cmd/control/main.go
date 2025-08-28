package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/alecthomas/kong"
	"github.com/go-ap/fedbox"
	"github.com/go-ap/fedbox/internal/cmd"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
)

var version = "HEAD"

func main() {
	if build, ok := debug.ReadBuildInfo(); ok && version == "HEAD" && build.Main.Version != "(devel)" && build.Main.Version != "" {
		version = build.Main.Version
	}

	CTLRun := new(cmd.CTL)
	ctx := kong.Parse(
		CTLRun,
		kong.Name("fedboxctl"),
		kong.Description("helper utility to manage ${name} instances."),
		kong.Vars{
			"version":            version,
			"name":               cmd.AppName,
			"defaultEnv":         string(env.DEV),
			"envTypes":           fmt.Sprintf("%s, %s, %s, %s", env.TEST, env.DEV, env.QA, env.PROD),
			"keyTypes":           fmt.Sprintf("%s, %s", fedbox.KeyTypeED25519, fedbox.KeyTypeRSA),
			"storageTypes":       fmt.Sprintf("%s, %s, %s, %s", config.StorageFS, config.StorageSqlite, config.StorageBoltDB, config.StorageBadger),
			"defaultKeyType":     fedbox.KeyTypeRSA,
			"defaultObjectTypes": fmt.Sprintf("%v", cmd.ValidGenericTypes),
		},
	)

	if err := ctx.Run(cmd.InitControl(CTLRun)); err != nil {
		cmd.Errf(err.Error())
		os.Exit(1)
	}
}
