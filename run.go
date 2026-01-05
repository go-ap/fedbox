package fedbox

import (
	"fmt"
	"io"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
)

const defaultWaitDuration = 5 * time.Second

var kongDefaultVars = kong.Vars{
	"version":            AppVersion,
	"name":               AppName,
	"defaultEnv":         string(env.DEV),
	"envTypes":           fmt.Sprintf("%s, %s, %s, %s", env.TEST, env.DEV, env.QA, env.PROD),
	"keyTypes":           fmt.Sprintf("%s, %s", ap.KeyTypeED25519, ap.KeyTypeRSA),
	"storageTypes":       fmt.Sprintf("%s, %s, %s, %s", config.StorageFS, config.StorageSqlite, config.StorageBoltDB, config.StorageBadger),
	"defaultKeyType":     ap.KeyTypeRSA,
	"defaultObjectTypes": fmt.Sprintf("%v", ValidGenericTypes),
}

func Run(args ...string) error {
	CTLRun := new(CTL)
	kongDefaultVars["defaultWaitDuration"] = defaultWaitDuration.String()

	k, err := kong.New(
		CTLRun,
		kong.Name("fedbox"),
		kong.Description("${name} server (version ${version})"),
		kongDefaultVars,
		kong.Bind(new(io.Writer)),
	)

	if err != nil {
		return err
	}
	ctx, err := k.Parse(args)
	if err != nil {
		return err
	}

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
