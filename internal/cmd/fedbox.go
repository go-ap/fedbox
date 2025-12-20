package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"git.sr.ht/~mariusor/lw"
	"git.sr.ht/~mariusor/storage-all"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox"
)

const AppName = "FedBOX"

type Run struct {
	Listen  string        `help:"The socket on which to listen" env:"FEDBOX_LISTEN" env:"LISTEN"`
	Wait    time.Duration `help:"The duration for which the server waits for existing connections to finish" default:"${defaultWaitDuration}"`
	Profile bool          `hidden:""`
}

func (r Run) Run(ctl *Control) error {
	var err error

	conf := ctl.Conf
	conf.TimeOut = r.Wait
	if r.Listen != "" {
		conf.Listen = r.Listen
	}

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
	db, err := storage.New(conf.StorageInitFns(l)...)
	if err != nil {
		l.Errorf("Unable to open storage: %s", err)
		return err
	}

	a, err := fedbox.New(l.WithContext(lw.Ctx{"log": "fedbox", "env": conf.Env}), conf, db)
	if err != nil {
		l.Errorf("Unable to initialize: %s", err)
		return err
	}

	return a.Run(context.Background())
}
