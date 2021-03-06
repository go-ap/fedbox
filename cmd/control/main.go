package main

import (
	"fmt"
	"github.com/go-ap/fedbox/internal/cmd"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
	"gopkg.in/urfave/cli.v2"
	"os"
)

var version = "HEAD"

func main() {
	app := cli.App{}
	app.Name = "fedbox-ctl"
	app.Usage = "helper utility to manage a FedBOX instance"
	app.Version = version
	app.Before = cmd.Before
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:  "url",
			Usage: "The url used by the application",
		},
		&cli.StringFlag{
			Name:  "env",
			Usage: fmt.Sprintf("The environment to use. Possible values: %q", []env.Type{env.DEV, env.QA, env.PROD}),
			Value: string(env.DEV),
		},
		&cli.StringFlag{
			Name:  "type",
			Usage: fmt.Sprintf("Type of the backend to use. Possible values: %q", []config.StorageType{config.StorageBoltDB, config.StorageBadger, config.StorageFS}),
		},
		&cli.StringFlag{
			Name:  "path",
			Value: ".",
			Usage: fmt.Sprintf("The path for the storage folder or socket"),
		},
		&cli.StringFlag{
			Name:  "user",
			Value: "fedbox",
			Usage: "The postgres database user",
		},
	}
	app.Commands = []*cli.Command{
		cmd.PubCmd,
		cmd.OAuth2Cmd,
		cmd.BootstrapCmd,
		cmd.AccountsCmd,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
