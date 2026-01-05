package cmd

import (
	"fmt"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
	"github.com/go-ap/fedbox"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
)

const defaultWaitDuration = 5 * time.Second

func Run() error {
	if build, ok := debug.ReadBuildInfo(); ok && AppVersion != "HEAD" && build.Main.Version != "(devel)" && build.Main.Version != "" {
		AppVersion = build.Main.Version
	}
	CTLRun := new(CTL)
	ctx := kong.Parse(
		CTLRun,
		kong.Name("fedbox"),
		kong.Description("${name} server (version ${version})"),
		kong.Vars{
			"defaultWaitDuration": defaultWaitDuration.String(),
			"version":             AppVersion,
			"name":                AppName,
			"defaultEnv":          string(env.DEV),
			"envTypes":            fmt.Sprintf("%s, %s, %s, %s", env.TEST, env.DEV, env.QA, env.PROD),
			"keyTypes":            fmt.Sprintf("%s, %s", fedbox.KeyTypeED25519, fedbox.KeyTypeRSA),
			"storageTypes":        fmt.Sprintf("%s, %s, %s, %s", config.StorageFS, config.StorageSqlite, config.StorageBoltDB, config.StorageBadger),
			"defaultKeyType":      fedbox.KeyTypeRSA,
			"defaultObjectTypes":  fmt.Sprintf("%v", ValidGenericTypes),
		},
	)

	ctl, err := InitControl(CTLRun, AppVersion)
	if err != nil {
		return err
	}

	switch ctx.Command() {
	case "maintenance", "stop", "reload", "run":
		// NOTE(marius): these don't interact with the storage, and additionally,
		// they involve sending their own signals, so we skip pausing.
	default:
		pauseFn := ctl.SendSignalToServer(syscall.SIGUSR1)
		_ = pauseFn()
		defer func() { _ = pauseFn() }()

		if err = ctl.Storage.Open(); err != nil {
			return err
		}
		defer ctl.Storage.Close()
	}

	return ctx.Run(ctl)
}
