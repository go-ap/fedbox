package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"git.sr.ht/~mariusor/lw"
	"github.com/alecthomas/kong"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
)

const AppName = "FedBOX"

type Run struct {
	Wait    time.Duration    `help:"The duration for which the server waits for existing connections to finish" default:"${defaultWaitDuration}"`
	Env     env.Type         `enum:"${envTypes}" help:"The environment to use. Expected values: ${envTypes}" default:"${defaultEnv}"`
	Path    string           `path:"" help:"The path for the storage folder." default:"." env:"STORAGE_PATH"`
	Profile bool             `hidden:""`
	Version kong.VersionFlag `short:"V"`
}

func (r Run) Run(version string) error {
	w := r.Wait
	e := r.Env

	conf, err := config.Load(r.Path, e, w)
	if err != nil {
		return err
	}

	conf.AppName = AppName
	conf.Profile = r.Profile
	conf.Secure = conf.Secure && !conf.Profile
	conf.Version = version

	var out io.WriteCloser
	if conf.LogOutput != "" {
		if out, err = os.Open(conf.LogOutput); err != nil {
			return errors.Newf("Unable to output logs to %s: %s", conf.LogOutput, err)
		}
		defer func() {
			if err := out.Close(); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "Unable to close log output: %s", err)
			}
		}()
	}
	var l lw.Logger
	if conf.Env.IsDev() {
		l = lw.Dev(lw.SetLevel(conf.LogLevel), lw.SetOutput(out))
	} else {
		l = lw.Prod(lw.SetLevel(conf.LogLevel), lw.SetOutput(out))
	}
	db, err := fedbox.Storage(conf, l.WithContext(lw.Ctx{"log": "storage"}))
	if err != nil {
		l.Errorf("Unable to open storage: %s", err)
		return err
	}

	a, err := fedbox.New(l.WithContext(lw.Ctx{"log": "fedbox", "env": e}), conf, db)
	if err != nil {
		l.Errorf("Unable to initialize: %s", err)
		return err
	}

	return a.Run(context.Background())
}
