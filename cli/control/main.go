package main

import (
	"bytes"
	"fmt"
	"github.com/go-ap/activitystreams"
	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/cmd"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
	"github.com/go-ap/fedbox/storage/boltdb"
	"github.com/go-ap/storage"
	"github.com/openshift/osin"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/urfave/cli.v2"
	"net/url"
	"os"
)

func setup(c *cli.Context, l logrus.FieldLogger, o *cmd.OAuth) error {
	dir := c.String("dir")
	if dir == "" {
		dir = "."
	}
	environ := env.Type(c.String("env"))
	if environ == "" {
		environ = env.DEV
	}
	typ := config.StorageType(c.String("type"))
	if typ == "" {
		typ = config.BoltDB
	}
	URI := c.String("url")
	if URI == "" {
		return errors.Newf("Missing url flag")
	}
	u, err := url.Parse(URI)
	if err != nil {
		l.Errorf("URL value passed is invalid: %S", err)
		return err
	}
	host := u.Hostname()
	var aDb osin.Storage
	var db storage.Repository
	if typ == config.BoltDB {
		path := config.GetBoltDBPath(dir, fmt.Sprintf("%s-oauth", host), environ)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			err := auth.BootstrapBoltDB(path, []byte(host))
			if err != nil {
				l.Errorf("Unable to create missing boltdb file %s: %s", path, err)
				return err
			}
		}

		aDb = auth.NewBoltDBStore(auth.Config{
			Path:       path,
			BucketName: host,
			LogFn:      func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Infof(s, p...) },
			ErrFn:      func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Errorf(s, p...) },
		})
		db = boltdb.New(boltdb.Config{
			Path:       config.GetBoltDBPath(dir, host, environ),
			BucketName: host,
			LogFn:      func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Infof(s, p...) },
			ErrFn:      func(f logrus.Fields, s string, p ...interface{}) { l.WithFields(f).Errorf(s, p...) },
		}, u.String())
	}
	if typ == config.Postgres {
		return errors.NotImplementedf("%s type not implemented", typ)
	}
	*o = cmd.NewOAuth(u, aDb, db)

	return nil
}

var version = "HEAD"

func main() {
	var command cmd.OAuth
	logger := logrus.New()
	logger.Level = logrus.ErrorLevel

	app := cli.App{}
	app.Name = "fedbox-ctl"
	app.Usage = "helper utility to manage a fedbox instance"
	app.Version = version
	app.Before = func(c *cli.Context) error {
		return nil
	}
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:  "url",
			Usage: "The url used by the application (REQUIRED)",
		},
		&cli.StringFlag{
			Name: "env",
			Usage: fmt.Sprintf("The environment to use. Possible values: %q, %q, %q", env.DEV, env.QA, env.PROD),
			Value: string(env.DEV),
		},
		&cli.StringFlag{
			Name: "type",
			Usage: fmt.Sprintf("Type of the backend to use. Possible values: %q, %q", config.BoltDB, config.Postgres),
			Value: string(config.BoltDB),
		},
		&cli.StringFlag{
			Name:  "path",
			Value: ".",
			Usage: "The folder where Bolt DBs",
		},
	}
	app.Commands = []*cli.Command{
		{
			Name:  "actor",
			Usage: "Actor management helper",
			Before: func(c *cli.Context) error {
				return setup(c, logger, &command)
			},
			Subcommands: []*cli.Command{
				{
					Name:    "add",
					Aliases: []string{"new"},
					Usage:   "Adds an ActivityPub actor",
					Action: func(c *cli.Context) error {
						names := c.Args().Slice()

						var actors = make(activitystreams.ItemCollection, 0)
						for _, name := range names {
							p, err := command.Ctl.AddActor(name, activitystreams.PersonType)
							if err != nil {
								fmt.Fprintf(os.Stderr, "Error adding %s: %s\n", name, err)
							}
							actors = append(actors, p)
						}
						return nil
					},
				},
			},
		},
		{
			Name: "oauth",
			Usage: "OAuth2 client and access token helper",
			Subcommands: []*cli.Command{
				{
					Name:  "client",
					Usage: "OAuth2 client application management",
					Before: func(c *cli.Context) error {
						return setup(c, logger, &command)
					},
					Subcommands: []*cli.Command{
						{
							Name:  "add",
							Aliases: []string{"new"},
							Usage: "Adds an OAuth2 client",
							Flags: []cli.Flag{
								&cli.StringSliceFlag{
									Name:  "redirectUri",
									Value: nil,
									Usage: "The redirect URIs for current application",
								},
							},
							Action: func(c *cli.Context) error {
								redirectURIs := c.StringSlice("redirectUri")
								if len(redirectURIs) < 1 {
									return errors.Newf("Need to provide at least a return URI for the client")
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
								id, err := command.AddClient(string(savpw), redirectURIs, nil)
								if err == nil {
									fmt.Sprintf("Client ID: %s", id)
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
								for i := 0; i <= c.Args().Len(); i++ {
									id := c.Args().Get(i)
									err := command.DeleteClient(id)
									if err != nil {
										fmt.Fprintf(os.Stderr, "Error deleting %s: %s\n", id, err)
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
								return errors.NotImplementedf("Client listing is not yet implemented")
							},
						},
					},
				},
				{
					Name:  "token",
					Usage: "OAuth2 authorization token management",
					Before: func(c *cli.Context) error {
						return setup(c, logger, &command)
					},
					Subcommands: []*cli.Command{
						{
							Name:  "add",
							Aliases: []string{"new", "get"},
							Usage: "Adds an OAuth2 token",
							Flags: []cli.Flag{
								&cli.StringFlag{
									Name:  "client",
									Usage: "The client to use for generating the token",
								},
								&cli.StringFlag{
									Name:  "handle",
									Usage: "The user handle we want to generate the authorization for",
								},
							},
							Action: func(c *cli.Context) error {
								clientID := c.String("client")
								if clientID == "" {
									return errors.Newf("Need to provide the client id")
								}
								handle := c.String("handle")
								if clientID == "" {
									return errors.Newf("Need to provide the actor handle")
								}
								tok, err := command.GenAuthToken(clientID, handle, nil)
								if err == nil {
									fmt.Printf("Authorization: Bearer %s\n", tok)
								}
								return err
							},
						},
					},
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
