package fedbox

import (
	"fmt"
	"io"
	"time"

	"github.com/alecthomas/kong"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
)

const defaultWaitDuration = 1 * time.Millisecond

var kongDefaultVars = kong.Vars{
	"version":             AppVersion,
	"name":                AppName,
	"defaultEnv":          string(env.DEV),
	"envTypes":            fmt.Sprintf("%s, %s, %s, %s", env.TEST, env.DEV, env.QA, env.PROD),
	"keyTypes":            fmt.Sprintf("%s, %s", ap.KeyTypeED25519, ap.KeyTypeRSA),
	"storageTypes":        fmt.Sprintf("%s, %s, %s, %s", config.StorageFS, config.StorageSqlite, config.StorageBoltDB, config.StorageBadger),
	"defaultKeyType":      string(ap.KeyTypeRSA),
	"defaultWaitDuration": defaultWaitDuration.String(),
	"defaultObjectTypes":  fmt.Sprintf("%v", ValidGenericTypes),
}

var CTLRun = new(CTL)

func Run(args ...string) error {
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

	ctl, err := InitControl(CTLRun)
	if err != nil {
		return err
	}
	if ctx.Command() != "run" {
		return ctl.runCommand(ctx)
	}

	return ctx.Run(ctl)
}
