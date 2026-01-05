package fedbox

import "syscall"

type Maintenance struct{}

func (m Maintenance) Run(ctl *Base) error {
	return ctl.SendSignal(syscall.SIGUSR1)
}

type Reload struct{}

func (m Reload) Run(ctl *Base) error {
	return ctl.SendSignal(syscall.SIGHUP)
}

type Stop struct{}

func (m Stop) Run(ctl *Base) error {
	return ctl.SendSignal(syscall.SIGTERM)
}
