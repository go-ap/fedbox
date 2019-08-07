package main

import (
	"fmt"
	"github.com/go-ap/auth"
	"github.com/go-ap/fedbox/cmd"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/log"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"os"
)

func setup(c *cli.Context, l logrus.FieldLogger, o *cmd.OauthCLI) error {
	path := c.String("path")
	host := c.String("host")
	typ := config.StorageType(c.String("type"))

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

						savpw := "yuh4ckm3?!"
						id, err := oauth.AddClient(savpw, c.StringSlice("redirectUri"))
						if err == nil {
							logger.Info(id)
						}
						return err
					},
				},
				{
					Name:  "del",
					Usage: "Removes an existing OAuth2 client",
					ArgsUsage: "APPLICATION_UUID...",
					Action: func(c *cli.Context) error {
						err := setup(c, logger, &oauth)
						if err != nil {
							return err
						}
						for id := range c.Args() {
							err := oauth.DeleteClient(string(id))
							if err != nil {
							}
						}
						return nil
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
