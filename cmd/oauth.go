package cmd

import (
	"encoding/json"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/oauth"
	"github.com/openshift/osin"
	"github.com/pborman/uuid"
	"strings"
	"time"
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

func (o *OauthCLI) AddClient(pw string, redirect []string, u interface{}) (string, error) {
	id := uuid.New()

	// TODO(marius): add a local Client struct that implements Client and ClientSecretMatcher interfaces with bcrypt support
	//   It could even be a struct composite from an activitystreams.Application + secret and callback properties
	userData, _ := json.Marshal(u)
	c := osin.DefaultClient{
		Id:          id,
		Secret:      pw,
		RedirectUri: strings.Join(redirect, oauth.URISeparator),
		UserData:    userData,
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

func (o *OauthCLI) GenAuthToken(clientID, handle string, dat interface{}) (string, error) {
	cl, err := o.DB.GetClient(clientID)
	if err != nil {
		return "", err
	}

	now := time.Now()

	aud := &osin.AuthorizeData{
		Client:      cl,
		CreatedAt:   now,
		ExpiresIn:   86400,
		RedirectUri: cl.GetRedirectUri(),
		State:       "state",
		UserData:    dat,
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
		UserData:      dat,
	}

	ad := &osin.AccessData{
		Client:        ar.Client,
		AuthorizeData: ar.AuthorizeData,
		AccessData:    ar.AccessData,
		ExpiresIn:     ar.Expiration,
		Scope:         ar.Scope,
		RedirectUri:   cl.GetRedirectUri(),
		CreatedAt:     now,
		UserData:      ar.UserData,
	}

	// generate access token
	ad.AccessToken, ad.RefreshToken, err = (&osin.AccessTokenGenDefault{}).GenerateAccessToken(ad, ar.GenerateRefresh)
	if err != nil {
		return "", err
	}
	// save access token
	if err = o.DB.SaveAccess(ad); err != nil {
		return "", err
	}

	return ad.AccessToken, nil
}
