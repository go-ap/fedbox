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

	keyGenerator func(act *vocab.Actor) error

	debugMode atomic.Bool

	out io.Writer
	err io.Writer
	in  io.Reader
}

type Serve struct {
	Listen  string        `help:"The socket on which to listen" env:"FEDBOX_LISTEN" env:"LISTEN"`
	Wait    time.Duration `help:"The duration for which the server waits for existing connections to finish" default:"${defaultWaitDuration}"`
	Profile bool          `hidden:""`
}

func (r Serve) Run(ctl *Base) error {
	fb, err := New(ctl)
	if err != nil {
		return errors.Annotatef(err, "unable to initialize service")
	}

	return fb.Run(context.Background())
}
