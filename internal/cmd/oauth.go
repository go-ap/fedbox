package cmd

import (
	"fmt"
	"strings"

	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox"
)

type Client struct {
	Add  AddClient `cmd:"" help:"Adds an OAuth2 client."`
	Del  DelClient `cmd:"" help:"Removes an existing OAuth2 client."`
	List LsClient  `cmd:"" help:"Lists existing OAuth2 clients."`
}

type OAuth struct {
	Client Client `cmd:"" help:"OAuth2 client application management."`
	Token  Token  `cmd:"" help:"OAuth2 authorization token management."`
}

type LsClient struct{}

func (l LsClient) Run(ctl *fedbox.Base) error {
	clients, err := ctl.ListClients()
	if err != nil {
		return err
	}
	for i, client := range clients {
		fmt.Printf("%d %s - %s\n", i, client.GetId(), strings.ReplaceAll(client.GetRedirectUri(), "\n", " :: "))
	}
	return nil
}

type DelClient struct {
	Client []string `arg:"" help:"Removes an existing OAuth2 client"`
}

func (d DelClient) Run(ctl *fedbox.Base) error {
	for _, id := range d.Client {
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

type AddClient struct {
	RedirectURIs []string `name:"redirect-uri" help:"The redirect URIs for current application"`
}

func (a AddClient) Run(ctl *fedbox.Base) error {
	redirectURIs := a.RedirectURIs
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

type Token struct {
	Add AddToken `cmd:"" help:"Adds an OAuth2 token"`
}

type AddToken struct {
	Client string `help:"The client to use for generating the token"`
	Actor  string `arg:"" help:"The actor identifier we want to generate the authorization for (ID)"`
}

func (a AddToken) Run(ctl *fedbox.Base) error {
	clientID := a.Client
	if clientID == "" {
		clientID = string(ctl.Service.GetLink())
	}
	actor := a.Actor
	if clientID == "" {
		return errors.Newf("Need to provide the actor identifier (ID)")
	}
	tok, err := ctl.GenAuthToken(clientID, actor, nil)
	if err == nil {
		fmt.Printf("Authorization: Bearer %s\n", tok)
	}
	return err
}
