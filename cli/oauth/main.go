package main

import (
	"bytes"
	"fmt"
	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/cmd"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/log"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"golang.org/x/crypto/ssh/terminal"
	"os"
)

func setup(c *cli.Context, l logrus.FieldLogger, o *cmd.OauthCLI) error {
	path := c.GlobalString("path")
	host := c.GlobalString("host")
	typ := config.StorageType(c.GlobalString("type"))

	if typ == "" {
		typ = config.BoltDB
	}
	if typ == config.BoltDB {
		o.DB = auth.NewBoltDBStore(auth.Config{
			Path:       path,
			BucketName: host,
			LogFn:      func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Infof(s, p...) },
			ErrFn:      func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Errorf(s, p...) },
		})
		defer o.DB.Close()
	}
	if typ == config.Postgres {
		// TODO(marius): finish the pgStore implementation
		o.DB = auth.NewPgDBStore(/*auth.Config{
			db:    nil,
			LogFn: func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Infof(s, p...) },
			ErrFn: func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Errorf(s, p...) },
		})*/)
		defer o.DB.Close()
	}

	fmt.Printf("%s[%d]: %#v\n", c.Command.Name, c.NArg(), c.Args())
	return nil
}

var version = "HEAD"

func main() {
	oauth := cmd.OauthCLI{}

	logger := log.New()
	
	app := cli.NewApp()
	app.Name = "oauth"
	app.Usage = "helper to add OAuth2 clients"
	app.Version = version
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name: "type",
			Usage: fmt.Sprintf("Type of the backend to use. Possible values: %s, %s", config.BoltDB, config.Postgres),
			Value: string(config.BoltDB),
		},
		cli.StringFlag{
			Name:  "path",
			Value: "",
			Usage: "The file path for the Bolt DB",
		},
		cli.StringFlag{
			Name:  "host",
			Value: "",
			Usage: "The hostname used by application as namespace",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:  "client",
			Usage: "OAuth2 client application management",
			Subcommands: []cli.Command{
				{
					Name:  "add",
					Aliases: []string{"new"},
					Usage: "Adds an OAuth2 client",
					Flags: []cli.Flag{
						cli.StringSliceFlag{
							Name:  "redirectUri",
							Value: nil,
							Usage: "The redirect URIs for current application",
						},
					},
					Action: func(c *cli.Context) error {
						err := setup(c, logger, &oauth)
						if err != nil {
							return err
						}

						fmt.Print("password: ")
						pw1, _ := terminal.ReadPassword(0)
						fmt.Println()
						fmt.Print("   again: ")
						savpw, _ := terminal.ReadPassword(0)
						fmt.Println()
						if !bytes.Equal(pw1, savpw) {
							return errors.Errorf("Passwords do not match")
						}
						id, err := oauth.AddClient(string(savpw), c.StringSlice("redirectUri"))
						if err == nil {
							logger.Info(id)
						}
						return err
					},
				},
				{
					Name:  "del",
					Aliases: []string{"delete", "remove"},
					Usage: "Removes an existing OAuth2 client",
					ArgsUsage: "APPLICATION_UUID...",
					Action: func(c *cli.Context) error {
						err := setup(c, logger, &oauth)
						if err != nil {
							return err
						}
						for _, id := range c.Args() {
							err := oauth.DeleteClient(id)
							if err != nil {
								logger.Errorf("Error deleting %s: %s", id, err)
							}
						}
						return nil
					},
				},
				{
					Name:  "ls",
					Aliases: []string{"list"},
					Usage: "Lists existing OAuth2 clients",
					Action: func(c *cli.Context) error {
						err := setup(c, logger, &oauth)
						if err != nil {
							return err
						}
						return errors.NotImplementedf("Client listing is not yet implemented")
					},
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		logger.Errorf("Error: %s", err)
		os.Exit(1)
	}
}
