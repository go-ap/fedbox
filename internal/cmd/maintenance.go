package cmd

import (
	"syscall"

	"github.com/go-ap/errors"
)

type Maintenance struct{}

func (m Maintenance) Run(ctl *Control) error {
	return sendSignalToServer(ctl, syscall.SIGUSR1)()
}

type Reload struct{}

func (m Reload) Run(ctl *Control) error {
	return sendSignalToServer(ctl, syscall.SIGHUP)()
}

type Stop struct{}

func (m Stop) Run(ctl *Control) error {
	return sendSignalToServer(ctl, syscall.SIGTERM)()
}

func sendSignalToServer(ctl *Control, sig syscall.Signal) func() error {
	return func() error {
		pid, err := ctl.Conf.ReadPid()
		if err != nil {
			return errors.Annotatef(err, "unable to read pid file")
		}
		return syscall.Kill(pid, sig)
	}
}
