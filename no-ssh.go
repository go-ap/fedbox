//go:build !ssh

package fedbox

import (
	"syscall"

	m "git.sr.ht/~mariusor/servermux"
	"github.com/alecthomas/kong"
)

func initSSHServer(_ *FedBOX) (m.Server, error) {
	return nil, nil
}

// runCommand when ssh isn't available as a connection mechanism, we pause the main FedBOX process
// to close storage access, then we operate our command from this process, and finally un-pause it.
func (ctl *Base) runCommand(ctx *kong.Context) error {
	pauseFn := ctl.SendSignalToServer(syscall.SIGUSR1)

	if err := ctl.Storage.Open(); err != nil {
		return err
	}
	defer ctl.Storage.Close()

	if err := pauseFn(); err == nil {
		defer func() { _ = pauseFn() }()
	}

	if ctx.Command() != "storage bootstrap" {
		if err := ctl.LoadServiceActor(); err != nil {
			return err
		}
	}
	return ctx.Run(ctl)
}
