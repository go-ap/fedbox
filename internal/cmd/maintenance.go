package cmd

import (
	"syscall"

	"github.com/go-ap/fedbox"
)

type Maintenance struct{}

func (m Maintenance) Run(ctl *fedbox.Base) error {
	return ctl.SendSignal(syscall.SIGUSR1)
}

type Reload struct{}

func (m Reload) Run(ctl *fedbox.Base) error {
	return ctl.SendSignal(syscall.SIGHUP)
}

type Stop struct{}

func (m Stop) Run(ctl *fedbox.Base) error {
	return ctl.SendSignal(syscall.SIGTERM)
}
