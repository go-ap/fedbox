package cmd

import (
	"encoding/json"
	"fmt"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/oauth"
	"github.com/go-ap/storage"
	"github.com/google/uuid"
	"github.com/openshift/osin"
	"net/url"
	"path"
	"strings"
	"time"
)

type OAuth struct {
	AuthDB osin.Storage
	Ctl    Control
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

func NewOAuth(baseURL *url.URL, authDB osin.Storage, actorDb storage.Repository) OAuth {
	return OAuth{
		AuthDB: authDB,
		Ctl: Control{
			BaseURL: baseURL,
			ActorDB: actorDb,
		},
	}
}

func (o *OAuth) AddClient(pw string, redirect []string, u interface{}) (string, error) {
	var id string

	app, err := o.Ctl.AddActor("oauth-client-app", as.ApplicationType)
	if err != nil {
		return "", err
	}

	id = path.Base(string(*app.GetID()))
	// TODO(marius): allow for updates of the application actor with incoming parameters for Icon, Summary, samd.
	app.PreferredUsername = as.NaturalLanguageValues{
		{
			Ref:   as.NilLangRef,
			Value: fmt.Sprintf("%s-%s", app.PreferredUsername.First().Value, id),
		},
	}
	app.Endpoints = nil
	app.Outbox = nil
	app.Liked = nil
	app.Likes = nil
	app.URL = as.IRI(redirect[0])

	o.Ctl.ActorDB.UpdateActor(app)
	if id == "" {
		id = uuid.New().String()
	}

	// TODO(marius): add a local Client struct that implements Client and ClientSecretMatcher interfaces with bcrypt support
	//   It could even be a struct composite from an as.Application + secret and callback properties
	userData, _ := json.Marshal(u)
	c := osin.DefaultClient{
		Id:          id,
		Secret:      pw,
		RedirectUri: strings.Join(redirect, oauth.URISeparator),
		UserData:    userData,
	}

	if saver, ok := o.AuthDB.(ClientSaver); ok {
		err = saver.CreateClient(&c)
	} else {
		err = errors.Newf("invalid OAuth2 client backend")
	}
	return id, err
}

func (o *OAuth) DeleteClient(uuid string) error {
	var err error

	if saver, ok := o.AuthDB.(ClientSaver); ok {
		err = saver.RemoveClient(uuid)
	} else {
		err = errors.Newf("invalid OAuth2 client backend")
	}
	// TODO(marius): add remove actor in the Control command

	return err
}

func (o *OAuth) ListClients() ([]osin.DefaultClient, error) {
	var err error

	if ls, ok := o.AuthDB.(ClientLister); ok {
		return ls.ListClients()
	} else {
		err = errors.Newf("invalid OAuth2 client backend")
	}

	return nil, err
}

func (o *OAuth) GenAuthToken(clientID, handle string, dat interface{}) (string, error) {
	cl, err := o.AuthDB.GetClient(clientID)
	if err != nil {
		return "", err
	}

	now := time.Now()

	f := activitypub.Filters{
		Name: []string{handle},
		Type: []as.ActivityVocabularyType{
			as.ActorType,
		},
	}
	list, cnt, err := o.Ctl.ActorDB.LoadActors(f)
	if err != nil {
		return "", err
	}
	if cnt == 0 {
		return "", errors.Newf("Handle not found")
	}
	actor, err := auth.ToPerson(list.First())
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
	// save access token
	if err = o.AuthDB.SaveAccess(ad); err != nil {
		return "", err
	}

	return ad.AccessToken, nil
}
