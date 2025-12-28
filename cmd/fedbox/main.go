package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox"
	"github.com/go-ap/fedbox/internal/cmd"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
)

var version = "HEAD"

const (
	Success int = iota
	Error
)

func main() {
	os.Exit(run())
}

const defaultWaitDuration = 5 * time.Second

func run() int {
	if build, ok := debug.ReadBuildInfo(); ok && version == "HEAD" && build.Main.Version != "(devel)" && build.Main.Version != "" {
		version = build.Main.Version
	}

	CTLRun := new(cmd.CTL)
	ctx := kong.Parse(
		CTLRun,
		kong.Name("fedbox"),
		kong.Description("${name} server (version ${version})"),
		kong.Vars{
			"defaultWaitDuration": defaultWaitDuration.String(),
			"version":             version,
			"name":                cmd.AppName,
			"defaultEnv":          string(env.DEV),
			"envTypes":            fmt.Sprintf("%s, %s, %s, %s", env.TEST, env.DEV, env.QA, env.PROD),
			"keyTypes":            fmt.Sprintf("%s, %s", fedbox.KeyTypeED25519, fedbox.KeyTypeRSA),
			"storageTypes":        fmt.Sprintf("%s, %s, %s, %s", config.StorageFS, config.StorageSqlite, config.StorageBoltDB, config.StorageBadger),
			"defaultKeyType":      fedbox.KeyTypeRSA,
			"defaultObjectTypes":  fmt.Sprintf("%v", cmd.ValidGenericTypes),
		},
	)

	ctl, err := cmd.InitControl(CTLRun, version)
	if err != nil {
		cmd.Errf(errors.Annotatef(err, "Unable to open FedBOX storage for path %q", CTLRun.Path).Error())
		return Error
	}

	switch ctx.Command() {
	case "maintenance", "stop", "reload", "run":
		// NOTE(marius): these don't interact with the storage, and additionally,
		// they involve sending their own signals, so we skip pausing.
	default:
		pauseFn := sendSignalToServer(ctl, syscall.SIGUSR1)
		_ = pauseFn()
		defer func() { _ = pauseFn() }()

		if err = ctl.Storage.Open(); err != nil {
			cmd.Errf(errors.Annotatef(err, "Unable to open FedBOX storage for path %s", ctl.Conf.StoragePath).Error())
			return Error
		}
		defer ctl.Storage.Close()
	}

	if err = ctx.Run(ctl); err != nil {
		cmd.Errf("Error: %s", err.Error())
		return Error
	}
	return Success
}

func sendSignalToServer(ctl *cmd.Control, sig syscall.Signal) func() error {
	return func() error {
		return ctl.SendSignal(sig)
	}
}
