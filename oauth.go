package fedbox

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/filters"
	json "github.com/go-ap/jsonld"
	"github.com/openshift/osin"
)

func (b *Base) DeleteClient(id string) error {
	iri := vocab.IRI(id)
	if _, err := iri.URL(); err != nil {
		iri = vocab.IRI(fmt.Sprintf("%s/%s/%s", b.Conf.BaseURL, filters.ActorsType, id))
	}
	err := b.DeleteObjects("Remove OAuth2 Client", nil, iri)
	if err != nil {
		return err
	}

	return b.Storage.RemoveClient(iri.String())
}

func (b *Base) ListClients() ([]osin.Client, error) {
	return b.Storage.ListClients()
}

func (b *Base) GenAuthToken(clientID, actorIdentifier string, _ any) (string, error) {
	if u, err := vocab.IRI(clientID).URL(); err == nil {
		clientID = filepath.Base(u.Path)
	}
	cl, err := b.Storage.GetClient(clientID)
	if err != nil {
		return "", err
	}

	now := time.Now().Truncate(time.Second).UTC()
	var f vocab.IRI
	if u, err := url.Parse(actorIdentifier); err == nil {
		f = vocab.IRI(u.String())
	} else {
		f = SearchActorsIRI(b.Service.ID, ByName(actorIdentifier), ByType(vocab.ActorTypes...))
	}
	maybeActors, err := b.Storage.Load(f.GetLink())
	if err != nil {
		return "", err
	}
	if vocab.IsNil(maybeActors) {
		return "", errors.NotFoundf("not found")
	}
	var actor vocab.Item
	err = vocab.OnActor(maybeActors, func(act *vocab.Actor) error {
		actor = act
		return nil
	})
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
	if err = b.Storage.SaveAuthorize(aud); err != nil {
		return "", err
	}
	// save access token
	if err = b.Storage.SaveAccess(ad); err != nil {
		return "", err
	}

	return ad.AccessToken, nil
}

const URISeparator = "\n"

func (b *Base) AddClient(pw []byte, redirectUris []string, u any) (string, error) {
	var id string

	self := ap.Self(vocab.IRI(b.Conf.BaseURL))
	now := time.Now().UTC()
	name := "oauth-client-app"
	urls := make(vocab.ItemCollection, 0)

	for i, redirectUri := range redirectUris {
		if u, err := url.ParseRequestURI(redirectUri); err == nil {
			u.Path = filepath.Clean(u.Path)
			name = u.Host
			curURL := u.String()
			redirectUris[i] = curURL

			u.Path = ""
			_ = urls.Append(vocab.IRI(u.String()), vocab.IRI(curURL))
		}
	}
	p := &vocab.Application{
		Type:              vocab.ApplicationType,
		AttributedTo:      self.GetLink(),
		Audience:          vocab.ItemCollection{vocab.PublicNS},
		Generator:         self.GetLink(),
		Published:         now,
		PreferredUsername: vocab.DefaultNaturalLanguage(name),
		URL:               urls,
	}
	app, err := b.AddActor(p, pw, self)
	if err != nil {
		return "", err
	}
	if metaSaver, ok := b.Storage.(MetadataStorage); ok {
		if err = AddKeyToItem(metaSaver, p, KeyTypeRSA); err != nil {
			b.Logger.Errorf("Error saving metadata for application %s: %s", name, err)
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
		RedirectUri: strings.Join(redirectUris, URISeparator),
		UserData:    userData,
	}

	return id, b.Storage.CreateClient(&d)
}
