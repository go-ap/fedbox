package app

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"path"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/assets"
	st "github.com/go-ap/fedbox/storage"
	"github.com/go-ap/handlers"
	"github.com/go-ap/processing"
	"github.com/openshift/osin"
	"github.com/sirupsen/logrus"
	"github.com/unrolled/render"
	"golang.org/x/oauth2"
)

type account struct {
	username string
	pw       string
	actor    *pub.Actor
}

type ClientStorage interface {
	ClientSaver
	ClientLister
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
	ListClients() ([]osin.Client, error)
	GetClient(id string) (osin.Client, error)
}

func (a account) IsLogged() bool {
	return a.actor != nil && a.actor.PreferredUsername.First().Value.String() == a.username
}

func (a *account) FromActor(p *pub.Actor) {
	a.username = p.PreferredUsername.First().String()
	a.actor = p
}

type authService struct {
	baseIRI pub.IRI
	genID   processing.IDGenerator
	storage fedboxStorage
	auth    *auth.Server
	logger  logrus.FieldLogger
}

const (
	meKey           = "me"
	redirectUriKey  = "redirect_uri"
	clientIdKey     = "client_id"
	responseTypeKey = "response_type"

	ID osin.AuthorizeRequestType = "id"
)

func (i authService) IsValidRequest(r *http.Request) bool {
	clientID, err := url.QueryUnescape(r.FormValue(clientIdKey))
	if err != nil {
		return false
	}
	clURL, err := url.ParseRequestURI(clientID)
	if err != nil || clURL.Host == "" || clURL.Scheme == "" {
		return false
	}
	return true
}

func IndieAuthClientActor(author pub.Item, url *url.URL) *pub.Actor {
	now := time.Now().UTC()
	preferredUsername := url.Host
	p := pub.Person{
		Type:         pub.ApplicationType,
		AttributedTo: author.GetLink(),
		Audience:     pub.ItemCollection{pub.PublicNS},
		Generator:    author.GetLink(),
		Published:    now,
		Summary: pub.NaturalLanguageValues{
			{pub.NilLangRef, pub.Content("IndieAuth generated actor")},
		},
		Updated: now,
		PreferredUsername: pub.NaturalLanguageValues{
			{pub.NilLangRef, pub.Content(preferredUsername)},
		},
		URL: pub.IRI(url.String()),
	}

	return &p
}

func filters(r *http.Request, baseURL pub.IRI) *activitypub.Filters {
	f, _ := activitypub.FromRequest(r, baseURL.String())
	f.IRI = f.IRI[:0]
	f.Collection = activitypub.ActorsType
	return f
}

func (i authService) ValidateClient(r *http.Request) (*pub.Actor, error) {
	r.ParseForm()
	clientID, err := url.QueryUnescape(r.FormValue(clientIdKey))
	if err != nil {
		return nil, err
	}
	if clientID == "" {
		return nil, nil
	}
	clientURL, err := url.Parse(clientID)
	if err != nil {
		return nil, nil
	}

	unescapedUri, err := url.QueryUnescape(r.FormValue(redirectUriKey))
	if err != nil {
		return nil, err
	}
	// load the me value of the actor that wants to authenticate
	me, err := url.QueryUnescape(r.FormValue(meKey))
	if err != nil {
		return nil, err
	}

	// check for existing user actor
	var actor pub.Item
	if me != "" {
		f := filters(r, i.baseIRI)
		f.Type = activitypub.CompStrs{activitypub.StringEquals(string(pub.PersonType))}
		f.URL = activitypub.CompStrs{activitypub.StringEquals(me)}
		actor, err = i.storage.repo.Load(f.GetLink())
		if err != nil {
			return nil, err
		}
		if actor == nil {
			return nil, errors.NotFoundf("unknown actor")
		}
	}

	// check for existing application actor
	f := filters(r, i.baseIRI)
	f.Type = activitypub.CompStrs{activitypub.StringEquals(string(pub.ApplicationType))}
	f.URL = activitypub.CompStrs{activitypub.StringEquals(clientID)}
	clientActor, err := i.storage.repo.Load(f.GetLink())
	if err != nil {
		return nil, err
	}
	if clientActor == nil {
		newClient := IndieAuthClientActor(actor, clientURL)
		if err != nil {
			return nil, err
		}
		if newId, err := i.genID(newClient, handlers.Outbox.IRI(actor), nil); err == nil {
			newClient.ID = newId
		}
		clientActor, err = i.storage.repo.Save(newClient)
		if err != nil {
			return nil, err
		}
	}
	id := path.Base(clientActor.GetID().String())
	// must have a valid client
	if _, err = i.storage.oauth.GetClient(id); err != nil {
		if errors.IsNotFound(err) {
			// create client
			newClient := osin.DefaultClient{
				Id:          id,
				Secret:      "",
				RedirectUri: unescapedUri,
				//UserData:    userData,
			}
			st, ok := i.storage.oauth.(ClientStorage)
			if !ok {
				return nil, errors.Errorf("Unable to create new client for IndieAuth request")
			}
			if err = st.CreateClient(&newClient); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	r.Form.Set(clientIdKey, id)
	if osin.AuthorizeRequestType(r.FormValue(responseTypeKey)) == ID {
		r.Form.Set(responseTypeKey, "code")
	}
	if act, ok := actor.(*pub.Actor); ok {
		return act, nil
	}
	return nil, nil
}

var scopeAnonymousUserCreate = "anonUserCreate"

func (i *authService) loadAccountFromPost(r *http.Request) (*account, error) {
	pw := r.PostFormValue("pw")
	handle := r.PostFormValue("handle")

	i.logger.WithFields(logrus.Fields{
		"handle": handle,
		"pass":   pw,
	}).Info("received")

	a := activitypub.Self(i.baseIRI)

	f := activitypub.FiltersNew()
	f.Name = activitypub.CompStrs{activitypub.CompStr{Str: handle}}
	f.IRI = activitypub.ActorsType.IRI(a)
	f.Type = activitypub.CompStrs{activitypub.StringEquals(string(pub.PersonType))}
	actors, err := i.storage.repo.Load(f.GetLink())
	if err != nil {
		return nil, errUnauthorized
	}

	var act *account
	if pwLoader, ok := i.storage.repo.(st.PasswordChanger); ok {
		if act, err = checkPw(actors, []byte(pw), pwLoader); err != nil {
			return nil, err
		}
	}
	return act, nil
}

func (i *authService) Authorize(w http.ResponseWriter, r *http.Request) {
	s := i.auth.Server
	resp := s.NewResponse()
	defer resp.Close()

	var err error
	actor := &auth.AnonymousActor
	if i.IsValidRequest(r) {
		if actor, err = i.ValidateClient(r); err != nil {
			resp.SetError(osin.E_INVALID_REQUEST, err.Error())
			redirectOrOutput(resp, w, r)
			return
		}
	}

	var overrideRedir = false

	if ar := s.HandleAuthorizeRequest(resp, r); ar != nil {
		if r.Method == http.MethodGet {
			if ar.Scope == scopeAnonymousUserCreate {
				// FIXME(marius): this seems like a way to backdoor our selves, we need a better way
				ar.Authorized = true
				overrideRedir = true
				iri := ar.HttpRequest.URL.Query().Get("actor")
				ar.UserData = iri
			} else {
				// this is basically the login page, with client being set
				m := login{title: "Login"}
				m.account = *actor
				m.client = ar.Client.GetId()
				m.state = ar.State

				i.renderTemplate(r, w, "login", m)
				return
			}
		} else {
			acc, err := i.loadAccountFromPost(r)
			if err != nil {
				errors.HandleError(err).ServeHTTP(w, r)
				return
			}
			if acc != nil {
				ar.Authorized = true
				ar.UserData = acc.actor.GetLink()
			}
		}
		s.FinishAuthorizeRequest(resp, r, ar)
	}
	if overrideRedir {
		resp.Type = osin.DATA
	}
	redirectOrOutput(resp, w, r)
}

func checkPw(it pub.Item, pw []byte, pwLoader st.PasswordChanger) (*account, error) {
	acc := new(account)
	found := false
	err := pub.OnActor(it, func(p *pub.Actor) error {
		if found {
			return nil
		}
		if err := pwLoader.PasswordCheck(p, pw); err == nil {
			acc.FromActor(p)
			found = true
		}
		return nil
	})
	if !found {
		return nil, errUnauthorized
	}
	return acc, err
}

func (i *authService) Token(w http.ResponseWriter, r *http.Request) {
	s := i.auth.Server
	resp := s.NewResponse()
	defer resp.Close()

	acc := &AnonymousAcct
	if ar := s.HandleAccessRequest(resp, r); ar != nil {
		actorFilters := activitypub.FiltersNew()
		switch ar.Type {
		case osin.PASSWORD:
			if u, _ := url.ParseRequestURI(ar.Username); u != nil {
				// NOTE(marius): here we send the full actor IRI as a username to avoid handler collisions
				actorFilters.IRI = pub.IRI(ar.Username)
			} else {
				actorFilters.IRI = activitypub.ActorsType.IRI(i.baseIRI)
				actorFilters.Name = activitypub.CompStrs{activitypub.StringEquals(ar.Username)}
			}
		case osin.AUTHORIZATION_CODE:
			if iri, ok := ar.UserData.(string); ok {
				actorFilters.IRI = pub.IRI(iri)
			}
		}
		actor, err := i.storage.repo.Load(actorFilters.GetLink())
		if err != nil {
			i.logger.Error(errUnauthorized)
			errors.HandleError(errUnauthorized).ServeHTTP(w, r)
			return
		}
		if ar.Type == osin.PASSWORD {
			if pwLoader, ok := i.storage.repo.(st.PasswordChanger); ok {
				if actor.IsCollection() {
					err = pub.OnCollectionIntf(actor, func(col pub.CollectionInterface) error {
						// NOTE(marius): This is a stupid way of doing pw authentication, as it will produce collisions
						//  for users with the same handle/pw and it will login the first in the collection.
						for _, actor := range col.Collection() {
							acc, err = checkPw(actor, []byte(ar.Password), pwLoader)
							if err == nil {
								return nil
							}
						}
						return errors.Newf("No actor matched the password")
					})
				} else {
					acc, err = checkPw(actor, []byte(ar.Password), pwLoader)
				}
				if err != nil || acc == nil {
					if err != nil {
						i.logger.Error(err)
					}
					errors.HandleError(errUnauthorized).ServeHTTP(w, r)
					return
				}
				ar.Authorized = acc.IsLogged()
				ar.UserData = acc.actor.GetLink()
			}
		}
		if ar.Type == osin.AUTHORIZATION_CODE {
			pub.OnActor(actor, func(p *pub.Actor) error {
				acc = new(account)
				acc.FromActor(p)
				ar.Authorized = acc.IsLogged()
				ar.UserData = acc.actor.GetLink()
				return nil
			})
		}
		s.FinishAccessRequest(resp, r, ar)
	}
	redirectOrOutput(resp, w, r)
}

func annotatedRsError(status int, old error, msg string, args ...interface{}) error {
	var err error
	switch status {
	case http.StatusForbidden:
		err = errors.NewForbidden(old, msg, args...)
	case http.StatusUnauthorized:
		err = errors.NewUnauthorized(old, msg, args...)
	case http.StatusInternalServerError:
		fallthrough
	default:
		err = errors.Annotatef(old, msg, args...)
	}

	return err
}

func redirectOrOutput(rs *osin.Response, w http.ResponseWriter, r *http.Request) {
	if rs.IsError {
		err := annotatedRsError(rs.StatusCode, rs.InternalError, "Error processing OAuth2 request: %s", rs.StatusText)
		errors.HandleError(err).ServeHTTP(w, r)
		return
	}
	// Add headers
	for i, k := range rs.Headers {
		for _, v := range k {
			w.Header().Add(i, v)
		}
	}

	if rs.Type == osin.REDIRECT {
		// Output redirect with parameters
		url, err := rs.GetRedirectUrl()
		if err != nil {
			err := annotatedRsError(http.StatusInternalServerError, err, "Error getting OAuth2 redirect URL")
			errors.HandleError(err).ServeHTTP(w, r)
			return
		}

		http.Redirect(w, r, url, http.StatusFound)
	} else {
		// set content type if the response doesn't already have one associated with it
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "application/json")
		}
		w.WriteHeader(rs.StatusCode)

		encoder := json.NewEncoder(w)
		if err := encoder.Encode(rs.Output); err != nil {
			errors.HandleError(err).ServeHTTP(w, r)
			return
		}
	}
}

type login struct {
	title   string
	account pub.Actor
	state   string
	client  string
}

func (l login) Title() string {
	return l.title
}

func (l login) Account() pub.Actor {
	return l.account
}

func (l login) State() string {
	return l.state
}

func (l login) Client() string {
	return l.client
}

func (l login) Handle() string {
	if len(l.account.PreferredUsername) == 0 {
		return ""
	}
	return l.account.PreferredUsername.First().String()
}

type model interface {
	Title() string
}

type authModel interface {
	model
	Account() pub.Actor
}

var (
	defaultRenderOptions = render.Options{
		AssetNames:                assets.Files,
		Asset:                     assets.Template,
		Directory:                 "templates/",
		Extensions:                []string{".html"},
		Funcs:                     []template.FuncMap{{"HTTPErrors": errors.HttpErrors}},
		Delims:                    render.Delims{Left: "{{", Right: "}}"},
		Charset:                   "UTF-8",
		DisableCharset:            false,
		BinaryContentType:         "application/octet-stream",
		HTMLContentType:           "text/html",
		IsDevelopment:             true,
		DisableHTTPErrorRendering: false,
	}
	errRenderer = render.New(defaultRenderOptions)
	ren = render.New(defaultRenderOptions)
)

func (i *authService) renderTemplate(r *http.Request, w http.ResponseWriter, name string, m authModel) error {
	err := ren.HTML(w, http.StatusOK, name, m)
	if err != nil {
		new := errors.Annotatef(err, "failed to render template")
		i.logger.WithFields(logrus.Fields{"template": name, "model": fmt.Sprintf("%T", m)}).Error(new.Error())
		errRenderer.HTML(w, http.StatusInternalServerError, "error", new)
	}

	return err
}

// ShowLogin serves GET /login requests
func (i *authService) ShowLogin(w http.ResponseWriter, r *http.Request) {
	a := activitypub.Self(i.baseIRI)

	m := login{title: "Login"}
	m.account = a

	i.renderTemplate(r, w, "login", m)
}

var errUnauthorized = errors.Unauthorizedf("Invalid username or password")

// HandleLogin handles POST /login requests
func (i *authService) HandleLogin(w http.ResponseWriter, r *http.Request) {
	acc, err := i.loadAccountFromPost(r)
	if err != nil {
		errors.HandleError(err).ServeHTTP(w, r)
		return
	}
	client := r.PostFormValue("client")
	state := r.PostFormValue("state")
	endpoints := pub.Endpoints{
		OauthAuthorizationEndpoint: pub.IRI(fmt.Sprintf("%s/oauth/authorize", i.baseIRI)),
		OauthTokenEndpoint:         pub.IRI(fmt.Sprintf("%s/oauth/token", i.baseIRI)),
	}
	if acc.actor != nil && acc.actor.Endpoints != nil {
		endpoints = *acc.actor.Endpoints
	}
	config := oauth2.Config{
		ClientID: client,
		Endpoint: oauth2.Endpoint{
			AuthURL:  endpoints.OauthAuthorizationEndpoint.GetLink().String(),
			TokenURL: endpoints.OauthTokenEndpoint.GetLink().String(),
		},
	}
	http.Redirect(w, r, config.AuthCodeURL(state, oauth2.AccessTypeOnline), http.StatusPermanentRedirect)
}

type OAuth struct {
	Provider     string
	Code         string
	Token        string
	RefreshToken string
	TokenType    string
	Expiry       time.Time
	State        string
}

type pwChange struct {
	title   string
	account pub.Actor
}

func (p pwChange) Title() string {
	return p.title
}

func (p pwChange) Account() pub.Actor {
	return p.account
}

// ShowChangePw
func (i *authService) ShowChangePw(w http.ResponseWriter, r *http.Request) {
	actor := i.loadActorFromOauth2Session(w, r)
	m := pwChange{
		title: "Change password",
	}
	if actor == nil {
		return
	}
	m.account = *actor

	i.renderTemplate(r, w, "password", m)
}

// HandleChangePw
func (i *authService) HandleChangePw(w http.ResponseWriter, r *http.Request) {
	actor := i.loadActorFromOauth2Session(w, r)
	if actor == nil {
		i.logger.Errorf("Unable to load actor from session")
		errors.HandleError(errors.NotValidf("Unable to load actor from session")).ServeHTTP(w, r)
		return
	}
	tok := r.URL.Query().Get("s")

	pw := r.PostFormValue("pw")
	pwConf := r.PostFormValue("pw-confirm")
	if pw != pwConf {
		errors.HandleError(errors.Newf("Different passwords submitted")).ServeHTTP(w, r)
		return
	}

	i.logger.WithFields(logrus.Fields{
		"handle": actor.PreferredUsername.String(),
		"pass":   pw,
	}).Info("received")

	if pwSetter, ok := i.storage.repo.(st.PasswordChanger); ok {
		err := pwSetter.PasswordSet(actor, []byte(pw))
		if err != nil {
			i.logger.Errorf("Error when saving password: %s", err)
			errors.HandleError(errors.NotValidf("Unable to change password")).ServeHTTP(w, r)
			return
		}
		i.storage.oauth.RemoveAuthorize(tok)
	}
}

func (i *authService) loadActorFromOauth2Session(w http.ResponseWriter, r *http.Request) *pub.Actor {
	notF := errors.NotFoundf("Not found")
	// TODO(marius): we land on this handler, coming from an email link containing a token identifying the Actor
	tok := r.URL.Query().Get("s")
	if len(tok) == 0 {
		i.logger.Errorf("Unable to load token from URL")
		errors.HandleError(notF).ServeHTTP(w, r)
		return nil
	}

	authSess, err := i.auth.Server.Storage.LoadAuthorize(tok)
	if err != nil {
		i.logger.Errorf("Error when loading authorize session: %s", err)
		errors.HandleError(notF).ServeHTTP(w, r)
		return nil
	}
	if authSess == nil {
		i.logger.Errorf("Invalid authorize session for tok %s", tok)
		errors.HandleError(notF).ServeHTTP(w, r)
		return nil
	}
	if authSess.ExpireAt().Sub(time.Now().UTC()) < 0 {
		i.logger.Errorf("Authorize token %s is expired %s", tok, authSess.ExpireAt().Format("2006-01-02 15:04:05"))
		errors.HandleError(notF).ServeHTTP(w, r)
		return nil
	}
	if authSess.UserData == nil {
		i.logger.Errorf("Invalid authorize session for tok %s, user-data is empty", tok)
		errors.HandleError(notF).ServeHTTP(w, r)
		return nil
	}

	actorIRI, err := assertToBytes(authSess.UserData)
	if err != nil {
		i.logger.Errorf("Invalid authorize session for tok %s, user-data is not an IRI: %v", tok, authSess.UserData)
		errors.HandleError(notF).ServeHTTP(w, r)
		return nil
	}
	ob, err := i.storage.repo.Load(pub.IRI(actorIRI))
	if err != nil || ob == nil {
		i.logger.Errorf("Error when loading actor from storage: %s", err)
		errors.HandleError(notF).ServeHTTP(w, r)
		return nil
	}
	var actor *pub.Actor
	pub.OnActor(ob, func(p *pub.Actor) error {
		actor = p
		return nil
	})
	return actor
}
func assertToBytes(in interface{}) ([]byte, error) {
	var ok bool
	var data string
	if in == nil {
		return nil, nil
	} else if data, ok = in.(string); ok {
		return []byte(data), nil
	} else if byt, ok := in.([]byte); ok {
		return byt, nil
	} else if byt, ok := in.(json.RawMessage); ok {
		return byt, nil
	} else if str, ok := in.(fmt.Stringer); ok {
		return []byte(str.String()), nil
	}
	return nil, errors.Errorf(`Could not assert "%v" to string`, in)
}
