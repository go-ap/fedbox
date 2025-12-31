package cmd

import (
	"context"
	"time"

	"git.sr.ht/~mariusor/lw"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox"
)

const AppName = "FedBOX"

type Run struct {
	Listen  string        `help:"The socket on which to listen" env:"FEDBOX_LISTEN" env:"LISTEN"`
	Wait    time.Duration `help:"The duration for which the server waits for existing connections to finish" default:"${defaultWaitDuration}"`
	Profile bool          `hidden:""`
}

func (r Run) Run(ctl *fedbox.Base) error {
	a, err := fedbox.New(ctl.Logger.WithContext(lw.Ctx{"log": "fedbox", "env": ctl.Conf.Env}), ctl.Conf, ctl.Storage)
	if err != nil {
		return errors.Annotatef(err, "unable to initialize service")
	}

	return a.Run(context.Background())
}
