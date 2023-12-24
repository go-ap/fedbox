package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox"
	ap "github.com/go-ap/fedbox/activitypub"
	s "github.com/go-ap/fedbox/storage"
	"github.com/go-ap/filters"
	"github.com/go-ap/processing"
	"github.com/openshift/osin"
	"github.com/urfave/cli/v2"
)

var client = &cli.Command{
	Name:  "client",
	Usage: "OAuth2 client application management",
	Subcommands: []*cli.Command{
		addClient,
		del,
		ls,
	},
}

var ls = &cli.Command{
	Name:    "ls",
	Aliases: []string{"list"},
	Usage:   "Lists existing OAuth2 clients",
	Action:  lsAct(&ctl),
}

func lsAct(c *Control) cli.ActionFunc {
	return func(c *cli.Context) error {
		clients, err := ctl.ListClients()
		if err != nil {
			return err
		}
		for i, client := range clients {
			fmt.Printf("%d %s - %s\n", i, client.GetId(), strings.ReplaceAll(client.GetRedirectUri(), "\n", " :: "))
		}
		return nil
	}
}

var del = &cli.Command{
	Name:      "del",
	Aliases:   []string{"delete", "remove", "rm"},
	Usage:     "Removes an existing OAuth2 client",
	ArgsUsage: "APPLICATION_UUID...",
	Action:    delAct(&ctl),
}

func delAct(c *Control) cli.ActionFunc {
	return func(c *cli.Context) error {
		for i := 0; i <= c.Args().Len(); i++ {
			id := c.Args().Get(i)
			if id == "" {
				continue
			}
			err := ctl.DeleteClient(id)
			if err != nil {
				Errf("Error deleting %s: %s\n", id, err)
				continue
			}
			fmt.Printf("Deleted: %s\n", id)
		}
		return nil
	}
}

var addClient = &cli.Command{
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
	Action: addAct(&ctl),
}

func addAct(c *Control) cli.ActionFunc {
	return func(c *cli.Context) error {
		redirectURIs := c.StringSlice("redirectUri")
		if len(redirectURIs) < 1 {
			return errors.Newf("Need to provide at least a redirect URI for the client")
		}
		pw, err := loadPwFromStdin(true, "client's")
		if err != nil {
			return err
		}
		id, err := ctl.AddClient(pw, redirectURIs, nil)
		if err == nil {
			fmt.Printf("Client ID: %s\n", id)
		}
		return err
	}
}

var token = &cli.Command{
	Name:        "token",
	Usage:       "OAuth2 authorization token management",
	Subcommands: []*cli.Command{tokenAdd},
}

var tokenAdd = &cli.Command{
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
	Action: tokenAct(&ctl),
}

var OAuth2Cmd = &cli.Command{
	Name:  "oauth",
	Usage: "OAuth2 client and access token helper",
	Subcommands: []*cli.Command{
		client,
		token,
	},
}

func tokenAct(c *Control) cli.ActionFunc {
	return func(c *cli.Context) error {
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
	}
}

const URISeparator = "\n"

func (c *Control) AddClient(pw []byte, redirect []string, u any) (string, error) {
	var id string

	self := ap.Self(vocab.IRI(ctl.Conf.BaseURL))
	now := time.Now().UTC()
	name := "oauth-client-app"
	urls := make(vocab.ItemCollection, 0)

	for i, redirectUrl := range redirect {
		if u, err := url.ParseRequestURI(redirectUrl); err == nil {
			u.Path = path.Clean(u.Path)
			name = u.Host
			curURL := u.String()
			redirect[i] = curURL

			u.Path = ""
			_ = urls.Append(vocab.IRI(u.String()), vocab.IRI(curURL))
		}
	}
	p := &vocab.Application{
		Type:              vocab.ApplicationType,
		AttributedTo:      self.GetLink(),
		Generator:         self.GetLink(),
		Published:         now,
		Updated:           now,
		PreferredUsername: vocab.DefaultNaturalLanguageValue(name),
		URL:               urls,
	}
	app, err := c.AddActor(p, pw, &self)
	if err != nil {
		return "", err
	}
	if metaSaver, ok := ctl.Storage.(s.MetadataTyper); ok {
		if err := AddKeyToItem(metaSaver, p, fedbox.KeyTypeRSA); err != nil {
			Errf("Error saving metadata for application %s: %s", name, err)
		}
	}

	id = path.Base(string(app.GetID()))
	// TODO(marius): allow for updates of the application actor with incoming parameters for Icon, Summary, samd.

	if id == "" {
		return "", errors.Newf("invalid actor saved, id is null")
	}

	// TODO(marius): add a local Client struct that implements Client and ClientSecretMatcher interfaces with bcrypt support
	//   It could even be a struct composite from an activitypub.Application + secret and callback properties
	userData, _ := json.Marshal(u)
	d := osin.DefaultClient{
		Id:          id,
		Secret:      string(pw),
		RedirectUri: strings.Join(redirect, URISeparator),
		UserData:    userData,
	}

	return id, c.Storage.CreateClient(&d)
}

func (c *Control) DeleteClient(uuid string) error {
	iri := fmt.Sprintf("%s/%s/%s", c.Conf.BaseURL, filters.ActorsType, uuid)
	err := c.DeleteObjects("Remove OAuth2 Client", nil, iri)
	if err != nil {
		return err
	}

	return c.Storage.RemoveClient(uuid)
}

func (c *Control) ListClients() ([]osin.Client, error) {
	return c.Storage.ListClients()
}

func (c *Control) GenAuthToken(clientID, actorIdentifier string, dat interface{}) (string, error) {
	if u, err := url.Parse(clientID); err == nil {
		clientID = path.Base(u.Path)
	}
	cl, err := c.Storage.GetClient(clientID)
	if err != nil {
		return "", err
	}

	now := time.Now().UTC()
	var f processing.Filterable
	if u, err := url.Parse(actorIdentifier); err == nil {
		f = vocab.IRI(u.String())
	} else {
		f = filters.FiltersNew(filters.Name(actorIdentifier), filters.Type(vocab.ActorTypes...))
	}
	list, err := c.Storage.Load(f.GetLink())
	if err != nil {
		return "", err
	}
	if vocab.IsNil(list) {
		return "", errors.NotFoundf("not found")
	}
	var actor vocab.Item
	if list.IsCollection() {
		err = vocab.OnCollectionIntf(list, func(c vocab.CollectionInterface) error {
			f := c.Collection().First()
			if f == nil {
				return errors.NotFoundf("no actor found %s", c.GetLink())
			}
			actor, err = vocab.ToActor(f)
			return err
		})
	} else {
		actor, err = vocab.ToActor(list)
	}
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
	if err = c.Storage.SaveAuthorize(aud); err != nil {
		return "", err
	}
	// save access token
	if err = c.Storage.SaveAccess(ad); err != nil {
		return "", err
	}

	return ad.AccessToken, nil
}
