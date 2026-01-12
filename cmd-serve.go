package fedbox

import (
	"context"
	"crypto"
	"io"
	"sync/atomic"
	"time"

	"git.sr.ht/~mariusor/lw"
	"git.sr.ht/~mariusor/storage-all"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/config"
)

const AppName = "FedBOX"

var AppVersion = "HEAD"

type Base struct {
	Conf              config.Options
	Logger            lw.Logger
	Service           vocab.Actor
	ServicePrivateKey crypto.PrivateKey
	Storage           storage.FullStorage

	debugMode atomic.Bool

	out io.Writer
	err io.Writer
	in  io.Reader
}

type Serve struct {
	Listen    string        `help:"The socket on which to listen" env:"FEDBOX_LISTEN" env:"LISTEN"`
	Wait      time.Duration `help:"The duration for which the server waits for existing connections to finish" default:"${defaultWaitDuration}"`
	Bootstrap bool          `hidden:""`
	Profile   bool          `hidden:""`
}

func (r Serve) Run(ctl *Base) error {
	a, err := New(ctl.Logger.WithContext(lw.Ctx{"env": ctl.Conf.Env}), ctl.Conf, ctl.Storage)
	if err != nil {
		return errors.Annotatef(err, "unable to initialize service")
	}

	if r.Bootstrap {
		if !a.Service.ID.Equal(vocab.IRI(ctl.Conf.BaseURL)) {
			keyType := ap.KeyTypeED25519
			if !ctl.Conf.MastodonIncompatible {
				keyType = ap.KeyTypeRSA
			}
			if err = a.Bootstrap(getPwAndKey(ctl, keyType)); err != nil {
				ctl.Logger.WithContext(lw.Ctx{"err": err}).Warnf("Unable to bootstrap service actor")
			}
		} else {
			ctl.Logger.Tracef("Bootstrap required but service actor exists")
		}
	}

	return a.Run(context.Background())
}
