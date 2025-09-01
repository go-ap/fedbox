package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"time"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox"
	ap "github.com/go-ap/fedbox/activitypub"
	s "github.com/go-ap/fedbox/storage"
	"github.com/go-ap/filters"
	"github.com/openshift/osin"
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

func (l LsClient) Run(ctl *Control) error {
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

func (d DelClient) Run(ctl *Control) error {
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

func (a AddClient) Run(ctl *Control) error {
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

func (a AddToken) Run(ctl *Control) error {
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

const URISeparator = "\n"

func (c *Control) AddClient(pw []byte, redirect []string, u any) (string, error) {
	var id string

	self := ap.Self(vocab.IRI(c.Conf.BaseURL))
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
	app, err := c.AddActor(p, pw, self)
	if err != nil {
		return "", err
	}
	if metaSaver, ok := c.Storage.(s.MetadataStorage); ok {
		if err := AddKeyToItem(metaSaver, p, fedbox.KeyTypeRSA); err != nil {
			Errf("Error saving metadata for application %s: %s", name, err)
		}
	}

	// TODO(marius): allow for updates of the application actor with incoming parameters for Icon, Summary, samd.

	id = app.GetID().String()
	if id == "" {
		return "", errors.Newf("invalid actor saved, id is null")
	}

	// TODO(marius): add a local Client struct that implements Client and ClientSecretMatcher interfaces with bcrypt support
	//   It could even be a struct composite from an vocab.Application + secret and callback properties
	userData, _ := json.Marshal(u)
	d := osin.DefaultClient{
		Id:          id,
		Secret:      string(pw),
		RedirectUri: strings.Join(redirect, URISeparator),
		UserData:    userData,
	}

	return id, c.Storage.CreateClient(&d)
}

func (c *Control) DeleteClient(id string) error {
	iri := vocab.IRI(id)
	if _, err := iri.URL(); err != nil {
		iri = vocab.IRI(fmt.Sprintf("%s/%s/%s", c.Conf.BaseURL, filters.ActorsType, id))
	}
	err := c.DeleteObjects("Remove OAuth2 Client", nil, iri)
	if err != nil {
		return err
	}

	return c.Storage.RemoveClient(iri.String())
}

func (c *Control) ListClients() ([]osin.Client, error) {
	return c.Storage.ListClients()
}

func (c *Control) GenAuthToken(clientID, actorIdentifier string, _ any) (string, error) {
	if u, err := vocab.IRI(clientID).URL(); err == nil {
		clientID = filepath.Base(u.Path)
	}
	cl, err := c.Storage.GetClient(clientID)
	if err != nil {
		return "", err
	}

	now := time.Now().UTC()
	var f vocab.IRI
	if u, err := url.Parse(actorIdentifier); err == nil {
		f = vocab.IRI(u.String())
	} else {
		f = SearchActorsIRI(c.Service.ID, ByName(actorIdentifier), ByType(vocab.ActorTypes...))
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

func ByName(names ...string) url.Values {
	q := make(url.Values)
	q["name"] = names
	return q
}

func ByType(types ...vocab.ActivityVocabularyType) url.Values {
	q := make(url.Values)
	tt := make([]string, len(types))
	for i, t := range types {
		tt[i] = string(t)
	}
	q["type"] = tt
	return q
}

func ByURL(urls ...vocab.IRI) url.Values {
	q := make(url.Values)
	uu := make([]string, len(urls))
	for i, u := range urls {
		uu[i] = u.String()
	}
	q["url"] = uu
	return q
}

func IRIWithFilters(iri vocab.IRI, searchParams ...url.Values) vocab.IRI {
	q := make(url.Values)
	for _, params := range searchParams {
		for k, vals := range params {
			if _, ok := q[k]; !ok {
				q[k] = make([]string, 0)
			}
			q[k] = append(q[k], vals...)
		}
	}
	if s, err := iri.URL(); err == nil {
		s.RawQuery = q.Encode()
		iri = vocab.IRI(s.String())
	}
	return iri
}

func SearchActorsIRI(baseIRI vocab.IRI, searchParams ...url.Values) vocab.IRI {
	return IRIWithFilters(filters.ActorsType.IRI(baseIRI), searchParams...)
}

func name(act *vocab.Actor) string {
	n := act.Name.First().String()
	if act.PreferredUsername != nil {
		n = act.PreferredUsername.First().String()
	}
	return n
}
