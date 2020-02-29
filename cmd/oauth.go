package cmd

import (
	"encoding/json"
	"fmt"
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	apub "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/storage"
	"github.com/openshift/osin"
	"github.com/pborman/uuid"
	"gopkg.in/urfave/cli.v2"
	"net/url"
	"path"
	"strings"
	"time"
)

type ClientSaver interface {
	// UpdateClient updates the client (identified by it's id) and replaces the values with the values of client.
	UpdateClient(c osin.Client) error
	// CreateClient stores the client in the database and returns an error, if something went wrong.
	CreateClient(c osin.Client) error
	// RemoveClient removes a client (identified by id) from the database. Returns an error if something went wrong.
	RemoveClient(id string) error
}

type ClientLister interface {
	// ListClients lists existing clients
	ListClients() ([]osin.Client, error)
}

func New(authDB osin.Storage, actorDb storage.Repository, conf config.Options) *Control {
	return &Control{
		BaseURL:     conf.BaseURL,
		Host:        conf.Host,
		Conf:        conf,
		AuthStorage: authDB,
		Storage:     actorDb,
	}
}

var OAuth2 = &cli.Command{
	Name:  "oauth",
	Usage: "OAuth2 client and access token helper",
	Subcommands: []*cli.Command{
		{
			Name:  "client",
			Usage: "OAuth2 client application management",
			Subcommands: []*cli.Command{
				{
					Name:    "add",
					Aliases: []string{"new"},
					Usage:   "Adds an OAuth2 client",
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
						pw, err := loadPwFromStdin(true, "client's")
						if err != nil {
							Errf(err.Error())
						}
						id, err := ctl.AddClient(pw, redirectURIs, nil)
						if err == nil {
							fmt.Printf("Client ID: %s\n", id)
						}
						return err
					},
				},
				{
					Name:      "del",
					Aliases:   []string{"delete", "remove", "rm"},
					Usage:     "Removes an existing OAuth2 client",
					ArgsUsage: "APPLICATION_UUID...",
					Action: func(c *cli.Context) error {
						for i := 0; i <= c.Args().Len(); i++ {
							id := c.Args().Get(i)
							err := ctl.DeleteClient(id)
							if err != nil {
								Errf("Error deleting %s: %s\n", id, err)
								continue
							}
							fmt.Printf("Deleted: %s\n", id)
						}
						return nil
					},
				},
				{
					Name:    "ls",
					Aliases: []string{"list"},
					Usage:   "Lists existing OAuth2 clients",
					Action: func(c *cli.Context) error {
						clients, err := ctl.ListClients()
						if err != nil {
							return err
						}
						for i, client := range clients {
							fmt.Printf("%d %s - %s\n", i, client.GetId(), strings.ReplaceAll(client.GetRedirectUri(), "\n", " :: "))
						}
						return nil
					},
				},
			},
		},
		{
			Name:  "token",
			Usage: "OAuth2 authorization token management",
			Subcommands: []*cli.Command{
				{
					Name:    "add",
					Aliases: []string{"new", "get"},
					Usage:   "Adds an OAuth2 token",
					Flags: []cli.Flag{
						&cli.StringFlag{
							Name:  "client",
							Usage: "The client to use for generating the token",
						},
						&cli.StringFlag{
							Name:  "actor",
							Usage: "The actor identifier we want to generate the authorization for (ID)",
						},
					},
					Action: func(c *cli.Context) error {
						clientID := c.String("client")
						if clientID == "" {
							return errors.Newf("Need to provide the client id")
						}
						actor := c.String("actor")
						if clientID == "" {
							return errors.Newf("Need to provide the actor identifier (ID)")
						}
						tok, err := ctl.GenAuthToken(clientID, actor, nil)
						if err == nil {
							fmt.Printf("Authorization: Bearer %s\n", tok)
						}
						return err
					},
				},
			},
		},
	},
}

const URISeparator = "\n"

func (c *Control) AddClient(pw []byte, redirect []string, u interface{}) (string, error) {
	var id string

	app, err := c.AddActor("oauth-client-app", pub.ApplicationType, nil, pw)
	if err != nil {
		return "", err
	}

	id = path.Base(string(app.GetID()))
	// TODO(marius): allow for updates of the application actor with incoming parameters for Icon, Summary, samd.
	app.PreferredUsername = pub.NaturalLanguageValues{
		{
			Ref:   pub.NilLangRef,
			Value: fmt.Sprintf("%s-%s", app.PreferredUsername.First().Value, id),
		},
	}
	app.Endpoints = nil
	app.Inbox = nil
	app.Liked = nil
	app.Likes = nil
	app.Followers = nil
	app.Following = nil
	app.URL = pub.IRI(redirect[0])

	c.Storage.UpdateActor(app)
	if id == "" {
		id = uuid.New()
	}

	// TODO(marius): add a local Client struct that implements Client and ClientSecretMatcher interfaces with bcrypt support
	//   It could even be a struct composite from an pub.Application + secret and callback properties
	userData, _ := json.Marshal(u)
	d := osin.DefaultClient{
		Id:          id,
		Secret:      string(pw),
		RedirectUri: strings.Join(redirect, URISeparator),
		UserData:    userData,
	}

	if saver, ok := c.AuthStorage.(ClientSaver); ok {
		err = saver.CreateClient(&d)
	} else {
		err = errors.Newf("invalid OAuth2 client backend")
	}
	return id, err
}

func (c *Control) DeleteClient(uuid string) error {
	c.DeleteActor(uuid)

	var err error
	if saver, ok := c.AuthStorage.(ClientSaver); ok {
		err = saver.RemoveClient(uuid)
	} else {
		err = errors.Newf("invalid OAuth2 client backend")
	}
	return err
}

func (c *Control) ListClients() ([]osin.Client, error) {
	var err error

	if ls, ok := c.AuthStorage.(ClientLister); ok {
		return ls.ListClients()
	} else {
		err = errors.Newf("invalid OAuth2 client backend")
	}

	return nil, err
}

func (c *Control) GenAuthToken(clientID, actorIdentifier string, dat interface{}) (string, error) {
	cl, err := c.AuthStorage.GetClient(clientID)
	if err != nil {
		return "", err
	}

	now := time.Now().UTC()
	var f storage.Filterable
	if u, err := url.Parse(actorIdentifier); err == nil {
		f = pub.IRI(u.String())
	} else {
		f = &apub.Filters{
			Name: apub.CompStrs{apub.CompStr{Str: actorIdentifier}},
			Type: pub.ActorTypes,
		}
	}
	list, cnt, err := c.Storage.LoadActors(f)
	if err != nil {
		return "", err
	}
	if cnt == 0 {
		return "", errors.Newf("Handle not found")
	}
	actor, err := pub.ToActor(list.First())
	if err != nil {
		return "", err
	}

	aud := &osin.AuthorizeData{
		Client:      cl,
		CreatedAt:   now,
		ExpiresIn:   86400,
		RedirectUri: cl.GetRedirectUri(),
		State:       "state",
	}

	// generate token code
	aud.Code, err = (&osin.AuthorizeTokenGenDefault{}).GenerateAuthorizeToken(aud)
	if err != nil {
		return "", err
	}

	// generate token directly
	ar := &osin.AccessRequest{
		Type:          osin.AUTHORIZATION_CODE,
		AuthorizeData: aud,
		Client:        cl,
		RedirectUri:   cl.GetRedirectUri(),
		Scope:         "scope",
		Authorized:    true,
		Expiration:    86400,
	}

	ad := &osin.AccessData{
		Client:        ar.Client,
		AuthorizeData: ar.AuthorizeData,
		AccessData:    ar.AccessData,
		ExpiresIn:     ar.Expiration,
		Scope:         ar.Scope,
		RedirectUri:   cl.GetRedirectUri(),
		CreatedAt:     now,
		UserData:      actor.GetLink(),
	}

	// generate access token
	ad.AccessToken, ad.RefreshToken, err = (&osin.AccessTokenGenDefault{}).GenerateAccessToken(ad, ar.GenerateRefresh)
	if err != nil {
		return "", err
	}
	// save authorize data
	if err = c.AuthStorage.SaveAuthorize(aud); err != nil {
		return "", err
	}
	// save access token
	if err = c.AuthStorage.SaveAccess(ad); err != nil {
		return "", err
	}

	return ad.AccessToken, nil
}
