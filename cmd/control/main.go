package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/go-ap/fedbox/internal/cmd"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
	"github.com/urfave/cli/v2"
)

var Version = "HEAD"

func main() {
	app := cli.App{}
	app.Name = "fedboxctl"
	app.Usage = "helper utility to manage a FedBOX instance"
	if build, ok := debug.ReadBuildInfo(); ok && Version == "HEAD" {
		app.Version = build.Main.Version
	}
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
		&cli.BoolFlag{
			Name:  "verbose",
			Usage: fmt.Sprintf("Increase verbosity level from the default associated with the environment settings."),
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
		cmd.FixStorageCollectionsCmd,
	}

	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
