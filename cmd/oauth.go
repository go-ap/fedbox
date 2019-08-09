package cmd

import (
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/oauth"
	"github.com/openshift/osin"
	"github.com/pborman/uuid"
	"strings"
)

type OauthCLI struct {
	DB osin.Storage
}

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
	ListClients() ([]osin.DefaultClient, error)
}

func (o *OauthCLI) AddClient(pw string, redirect []string) (string, error) {
	id := uuid.New()
	c := osin.DefaultClient{
		Id:          id,
		Secret:      pw,
		RedirectUri: strings.Join(redirect, oauth.URISeparator),
		UserData:    "{}",
	}

	var err error
	if saver, ok := o.DB.(ClientSaver); ok {
		err = saver.CreateClient(&c)
	} else {
		err = errors.Newf("invalid OAuth2 client backend")
	}

	return id, err
}

func (o *OauthCLI) DeleteClient(uuid string) error {
	var err error

	if saver, ok := o.DB.(ClientSaver); ok {
		err = saver.RemoveClient(uuid)
	} else {
		err = errors.Newf("invalid OAuth2 client backend")
	}

	return err
}

func (o *OauthCLI) ListClients() ([]osin.DefaultClient, error) {
	var err error

	if ls, ok := o.DB.(ClientLister); ok {
		return ls.ListClients()
	} else {
		err = errors.Newf("invalid OAuth2 client backend")
	}

	return nil, err
}
