package fedbox

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"path"
	"time"

	"git.sr.ht/~mariusor/lw"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/assets"
	st "github.com/go-ap/fedbox/storage"
	"github.com/go-ap/processing"
	"github.com/go-chi/chi/v5"
	"github.com/mariusor/render"
	"github.com/openshift/osin"
	"golang.org/x/oauth2"
)

type account struct {
	username string
	pw       string
	actor    *vocab.Actor
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

func (a *account) FromActor(p *vocab.Actor) {
	a.username = p.PreferredUsername.First().String()
	a.actor = p
}

type authService struct {
	baseIRI vocab.IRI
	genID   processing.IDGenerator
	storage fedboxStorage
	auth    *auth.Server
	logger  lw.Logger
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

func IndieAuthClientActor(author vocab.Item, url *url.URL) *vocab.Actor {
	now := time.Now().UTC()
	preferredUsername := url.Host
	p := vocab.Person{
		Type:         vocab.ApplicationType,
		AttributedTo: author.GetLink(),
		Audience:     vocab.ItemCollection{vocab.PublicNS},
		Generator:    author.GetLink(),
		Published:    now,
		Summary: vocab.NaturalLanguageValues{
			{vocab.NilLangRef, vocab.Content("IndieAuth generated actor")},
		},
		Updated: now,
		PreferredUsername: vocab.NaturalLanguageValues{
			{vocab.NilLangRef, vocab.Content(preferredUsername)},
		},
		URL: vocab.IRI(url.String()),
	}

	return &p
}

func filters(r *http.Request, baseURL vocab.IRI) *activitypub.Filters {
	f := activitypub.FromRequest(r, baseURL.String())
	f.IRI = f.IRI[:0]
	f.Collection = activitypub.ActorsType
	return f
}

func (i authService) ValidateClient(r *http.Request) (*vocab.Actor, error) {
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
	var actor vocab.Item
	if me != "" {
		f := filters(r, i.baseIRI)
		f.Type = activitypub.CompStrs{activitypub.StringEquals(string(vocab.PersonType))}
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
	f.Type = activitypub.CompStrs{activitypub.StringEquals(string(vocab.ApplicationType))}
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
		if newId, err := i.genID(newClient, vocab.Outbox.IRI(actor), nil); err == nil {
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
	if act, ok := actor.(*vocab.Actor); ok {
		return act, nil
	}
	return nil, nil
}

var scopeAnonymousUserCreate = "anonUserCreate"

func (i *authService) loadAccountByID(id string) (*vocab.Actor, error) {
	f := activitypub.FiltersNew()

	a := activitypub.Self(i.baseIRI)

	f.IRI = activitypub.ActorsType.IRI(a).AddPath(id)
	actors, err := i.storage.repo.Load(f.GetLink())
	if err != nil {
		return nil, err
	}
	if actors == nil {
		return nil, errNotFound
	}

	var actor *vocab.Actor
	err = vocab.OnActor(actors, func(act *vocab.Actor) error {
		actor = act
		return nil
	})
	if err != nil || actor == nil {
		return nil, errNotFound
	}
	return actor, nil
}

func (i *authService) loadAccountFromPost(r *http.Request) (*account, error) {
	pw := r.PostFormValue("pw")
	handle := r.PostFormValue("handle")

	i.logger.WithContext(lw.Ctx{
		"handle": handle,
		"pass":   pw,
	}).Infof("received")

	a := activitypub.Self(i.baseIRI)

	f := activitypub.FiltersNew()
	f.Name = activitypub.CompStrs{activitypub.CompStr{Str: handle}}
	f.IRI = activitypub.ActorsType.IRI(a)
	f.Type = activitypub.CompStrs{activitypub.StringEquals(string(vocab.PersonType))}
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

func checkPw(it vocab.Item, pw []byte, pwLoader st.PasswordChanger) (*account, error) {
	acc := new(account)
	found := false
	err := vocab.OnActor(it, func(p *vocab.Actor) error {
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
				actorFilters.IRI = vocab.IRI(ar.Username)
			} else {
				actorFilters.IRI = activitypub.ActorsType.IRI(i.baseIRI)
				actorFilters.Name = activitypub.CompStrs{activitypub.StringEquals(ar.Username)}
			}
		case osin.AUTHORIZATION_CODE:
			if iri, ok := ar.UserData.(string); ok {
				actorFilters.IRI = vocab.IRI(iri)
			}
		}
		actor, err := i.storage.repo.Load(actorFilters.GetLink())
		if err != nil {
			i.logger.Errorf("%s", errUnauthorized)
			errors.HandleError(errUnauthorized).ServeHTTP(w, r)
			return
		}
		if ar.Type == osin.PASSWORD {
			if pwLoader, ok := i.storage.repo.(st.PasswordChanger); ok {
				if actor.IsCollection() {
					err = vocab.OnCollectionIntf(actor, func(col vocab.CollectionInterface) error {
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
						i.logger.Errorf("%s", err)
					}
					errors.HandleError(errUnauthorized).ServeHTTP(w, r)
					return
				}
				ar.Authorized = acc.IsLogged()
				ar.UserData = acc.actor.GetLink()
			}
		}
		if ar.Type == osin.AUTHORIZATION_CODE {
			vocab.OnActor(actor, func(p *vocab.Actor) error {
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
	account vocab.Actor
	state   string
	client  string
}

func (l login) Title() string {
	return l.title
}

func (l login) Account() vocab.Actor {
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
	Account() vocab.Actor
}

var (
	defaultRenderOptions = render.Options{
		FileSystem:                assets.Templates,
		Directory:                 assets.TemplatesPath,
		Extensions:                []string{".html"},
		Funcs:                     []template.FuncMap{{"HTTPErrors": errors.HttpErrors}},
		Delims:                    render.Delims{Left: "{{", Right: "}}"},
		Charset:                   "UTF-8",
		DisableCharset:            false,
		HTMLContentType:           "text/html",
		DisableHTTPErrorRendering: false,
	}
	errRenderer = render.New(defaultRenderOptions)
	ren         = render.New(defaultRenderOptions)
)

func (i *authService) renderTemplate(r *http.Request, w http.ResponseWriter, name string, m authModel) {
	if err := ren.HTML(w, http.StatusOK, name, m); err != nil {
		new := errors.Annotatef(err, "failed to render template")
		i.logger.WithContext(lw.Ctx{"template": name, "model": fmt.Sprintf("%T", m)}).Errorf(new.Error())
		errRenderer.HTML(w, http.StatusInternalServerError, "error", new)
	}
}

func name(act *vocab.Actor) string {
	n := act.Name.First().String()
	if act.PreferredUsername != nil {
		n = act.PreferredUsername.First().String()
	}
	return n
}

// ShowLogin serves GET /login requests
func (i *authService) ShowLogin(w http.ResponseWriter, r *http.Request) {
	tit := "Login to FedBOX"
	m := login{title: tit}

	if id := chi.URLParam(r, "id"); id != "" {
		actor, err := i.loadAccountByID(id)
		if err != nil {
			errors.HandleError(err).ServeHTTP(w, r)
			return
		}
		// NOTE(marius): we allow only actors to login using oauth page
		if actor.Type != vocab.PersonType {
			errors.HandleError(errNotFound).ServeHTTP(w, r)
			return
		}

		m.account = *actor
		tit = fmt.Sprintf("Login to FedBOX as %s", name(actor))
	}

	if clientId := r.FormValue("client"); len(clientId) > 0 {
		app, err := i.loadAccountByID(clientId)
		if err != nil {
			errors.HandleError(activitypub.ErrNotFound("client application not found")).ServeHTTP(w, r)
			return
		}
		if app.Type == vocab.ApplicationType {
			m.client = clientId
		}
	}

	i.renderTemplate(r, w, "login", m)
}

var (
	errUnauthorized = errors.Unauthorizedf("Invalid username or password")
	errNotFound     = activitypub.ErrNotFound("actor not found")
)

// HandleLogin handles POST /login requests
func (i *authService) HandleLogin(w http.ResponseWriter, r *http.Request) {
	acc, err := i.loadAccountFromPost(r)
	if err != nil {
		errors.HandleError(err).ServeHTTP(w, r)
		return
	}
	client := r.PostFormValue("client")
	state := r.PostFormValue("state")
	endpoints := vocab.Endpoints{
		OauthAuthorizationEndpoint: vocab.IRI(fmt.Sprintf("%s/oauth/authorize", i.baseIRI)),
		OauthTokenEndpoint:         vocab.IRI(fmt.Sprintf("%s/oauth/token", i.baseIRI)),
	}
	if !vocab.IsNil(acc.actor) && acc.actor.Endpoints != nil {
		if acc.actor.Endpoints.OauthTokenEndpoint != nil {
			endpoints.OauthTokenEndpoint = acc.actor.Endpoints.OauthTokenEndpoint
		}
		if acc.actor.Endpoints.OauthAuthorizationEndpoint != nil {
			endpoints.OauthAuthorizationEndpoint = acc.actor.Endpoints.OauthAuthorizationEndpoint
		}
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
	account vocab.Actor
}

func (p pwChange) Title() string {
	return p.title
}

func (p pwChange) Account() vocab.Actor {
	return p.account
}

// ShowChangePw
func (i *authService) ShowChangePw(w http.ResponseWriter, r *http.Request) {
	actor := i.loadActorFromOauth2Session(w, r)
	if actor == nil {
		errors.HandleError(errors.NotValidf("Unable to load actor from session")).ServeHTTP(w, r)
		return
	}

	if id := chi.URLParam(r, "id"); id != "" {
		act, err := i.loadAccountByID(id)
		if err != nil {
			errors.HandleError(err).ServeHTTP(w, r)
			return
		}
		if !act.GetID().Equals(actor.GetID(), true) {
			errors.HandleError(errors.NotValidf("Unable to load actor from session")).ServeHTTP(w, r)
			return
		}
	}

	m := pwChange{
		title:   "Change password",
		account: *actor,
	}

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

	i.logger.WithContext(lw.Ctx{
		"handle": actor.PreferredUsername.String(),
		"pass":   pw,
	}).Infof("received")

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

func (i *authService) loadActorFromOauth2Session(w http.ResponseWriter, r *http.Request) *vocab.Actor {
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
	ob, err := i.storage.repo.Load(vocab.IRI(actorIRI))
	if err != nil || ob == nil {
		i.logger.Errorf("Error when loading actor from storage: %s", err)
		errors.HandleError(notF).ServeHTTP(w, r)
		return nil
	}
	var actor *vocab.Actor
	vocab.OnActor(ob, func(p *vocab.Actor) error {
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
