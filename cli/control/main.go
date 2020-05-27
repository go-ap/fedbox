package main

import (
	"fmt"
	"github.com/go-ap/fedbox/cmd"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
	"gopkg.in/urfave/cli.v2"
	"os"
)

type pgFlags struct {
	host string
	port int64
	user string
	pw   []byte
}

type boltFlags struct {
	path string
	root string
}

type ctlFlags struct {
	env      env.Type
	dir      string
	typ      config.StorageType
	url      string
	postgres pgFlags
	bolt     boltFlags
}

var version = "HEAD"

func main() {
	app := cli.App{}
	app.Name = "fedbox-ctl"
	app.Usage = "helper utility to manage a fedbox instance"
	app.Version = version
	app.Before = cmd.Before
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:  "url",
			Usage: "The url used by the application (REQUIRED)",
		},
		&cli.StringFlag{
			Name:  "env",
			Usage: fmt.Sprintf("The environment to use. Possible values: %q", []env.Type{env.DEV, env.QA, env.PROD}),
			Value: string(env.DEV),
		},
		&cli.StringFlag{
			Name:  "type",
			Usage: fmt.Sprintf("Type of the backend to use. Possible values: %q", []config.StorageType{config.BoltDB, config.Badger, config.FS}),
		},
		&cli.StringFlag{
			Name:  "path",
			Value: ".",
			Usage: fmt.Sprintf("The path for the storage folder orsocket"),
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

	err := app.Run(os.Args)
	if err != nil {
		cmd.Errf("Error: %s\n", err)
		os.Exit(1)
	}
}
