package cmd

import (
	"syscall"

	"github.com/urfave/cli/v2"
)

var Maintenance = &cli.Command{
	Name:   "maintenance",
	Usage:  "Toggle maintenance mode for the main FedBOX server",
	Action: sendSignalToServerAct(&ctl, syscall.SIGUSR1),
}

var Reload = &cli.Command{
	Name:   "reload",
	Usage:  "Reload the main FedBOX server configuration",
	Action: sendSignalToServerAct(&ctl, syscall.SIGHUP),
}

var Stop = &cli.Command{
	Name:   "stop",
	Usage:  "Stops the main FedBOX server configuration",
	Action: sendSignalToServerAct(&ctl, syscall.SIGTERM),
}

func sendSignalToServerAct(ctl *Control, sig syscall.Signal) cli.ActionFunc {
	return func(c *cli.Context) error {
		pid, err := ctl.Conf.ReadPid()
		if err != nil {
			return err
		}
		return syscall.Kill(pid, sig)
	}
}
