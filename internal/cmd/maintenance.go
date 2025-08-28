package cmd

import (
	"syscall"

	"github.com/go-ap/errors"
)

type Maintenance struct{}

func (m Maintenance) Run(ctl *Control) error {
	return ctl.SendSignal(syscall.SIGUSR1)
}

type Reload struct{}

func (m Reload) Run(ctl *Control) error {
	return ctl.SendSignal(syscall.SIGHUP)
}

type Stop struct{}

func (m Stop) Run(ctl *Control) error {
	return ctl.SendSignal(syscall.SIGTERM)
}

func (c *Control) SendSignal(sig syscall.Signal) error {
	pid, err := c.Conf.ReadPid()
	if err != nil {
		return errors.Annotatef(err, "unable to read pid file")
	}
	return syscall.Kill(pid, sig)
}
