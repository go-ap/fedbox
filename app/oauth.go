package app

import (
	"encoding/json"
	"fmt"
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/activitypub"
	st "github.com/go-ap/fedbox/storage"
	"github.com/go-ap/storage"
	"github.com/openshift/osin"
	"github.com/sirupsen/logrus"
	"github.com/unrolled/render"
	"golang.org/x/oauth2"
	"html/template"
	"net/http"
	"time"
)

type account struct {
	username string
	pw       string
	actor    *pub.Actor
}

func (a account) IsLogged() bool {
	return a.actor != nil && a.actor.PreferredUsername.First().Value == a.username
}

func (a *account) FromActor(p *pub.Actor) {
	a.username = p.PreferredUsername.First().String()
	a.actor = p
}

type oauthHandler struct {
	baseURL string
	os      *osin.Server
	loader  storage.ActorLoader
	logger  logrus.FieldLogger
}

var scopeAnonymousUserCreate = "anonUserCreate"

func (h *oauthHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	s := h.os

	acc := AnonymousAcct
	if actor, ok := auth.ActorContext(r.Context()); ok {
		acc.actor = &actor
	}

	resp := s.NewResponse()
	defer resp.Close()

	var overrideRedir = false
	if ar := s.HandleAuthorizeRequest(resp, r); ar != nil {
		if acc.IsLogged() {
			ar.Authorized = true
			ar.UserData = acc.actor.GetLink()
		} else {
			if ar.Scope == scopeAnonymousUserCreate {
				// FIXME(marius): this seems like a way to backdoor our selves, we need a better way
				ar.Authorized = true
				overrideRedir = true
				iri := ar.HttpRequest.URL.Query().Get("actor")
				ar.UserData = iri
			} else {
				m := login{title: "Login"}
				m.account = auth.AnonymousActor
				m.client = ar.Client.GetId()
				m.state = ar.State

				h.renderTemplate(r, w, "login", m)
				return
			}
		}
		s.FinishAuthorizeRequest(resp, r, ar)
	}
	if overrideRedir {
		resp.Type = osin.DATA
	}
	redirectOrOutput(resp, w, r)
}

func checkPw(it pub.Item, pw []byte, pwLoader st.PasswordChanger) (account, error) {
	acc := account{}
	err := pub.OnActor(it, func(p *pub.Actor) error {
		err := pwLoader.PasswordCheck(p, pw)
		if err != nil {
			// TODO(marius): log the received error
			return errUnauthorized
		}
		acc.FromActor(p)
		return nil
	})
	if err != nil {
		return acc, err
	}
	return acc, nil
}

func (h *oauthHandler) Token(w http.ResponseWriter, r *http.Request) {
	s := h.os
	resp := s.NewResponse()
	defer resp.Close()

	acc := AnonymousAcct
	if ar := s.HandleAccessRequest(resp, r); ar != nil {
		actorFilters := activitypub.FiltersNew()
		switch ar.Type {
		case osin.PASSWORD:
			actorFilters.IRI = activitypub.ActorsType.IRI(pub.IRI(h.baseURL))
			actorFilters.Name = activitypub.CompStrs{activitypub.CompStr{Str: ar.Username}}
		case osin.AUTHORIZATION_CODE:
			if iri, ok := ar.UserData.(string); ok {
				actorFilters.IRI = pub.IRI(iri)
			}
		}
		actors, _, err := h.loader.LoadActors(actorFilters)
		if err != nil {
			h.logger.Error(errUnauthorized)
			errors.HandleError(errUnauthorized).ServeHTTP(w, r)
			return
		}
		if ar.Type == osin.PASSWORD {
			if pwLoader, ok := h.loader.(st.PasswordChanger); ok {
				found := false
				for _, actor := range actors {
					acc, err = checkPw(actor, []byte(ar.Password), pwLoader)
					if err == nil {
						ar.Authorized = acc.IsLogged()
						ar.UserData = acc.actor.GetLink()
						found = true
						break
					}
				}
				if !found {
					h.logger.Error(errUnauthorized)
					errors.HandleError(errUnauthorized).ServeHTTP(w, r)
					return
				}
			}
		}
		if ar.Type == osin.AUTHORIZATION_CODE && len(actors) == 1 {
			pub.OnActor(actors.First(), func(p *pub.Actor) error {
				acc = account{}
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

type model interface {
	Title() string
}

type authModel interface {
	model
	Account() pub.Actor
}

var errRenderer = render.New(render.Options{
	Directory:  "templates/",
	Extensions: []string{".html"},
	Funcs: []template.FuncMap{{
		"HTTPErrors": func(err error) []errors.Http { _, errs := errors.HttpErrors(err); return errs },
	}},
	Delims:                    render.Delims{Left: "{{", Right: "}}"},
	Charset:                   "UTF-8",
	DisableCharset:            false,
	BinaryContentType:         "application/octet-stream",
	HTMLContentType:           "text/html",
	IsDevelopment:             true,
	DisableHTTPErrorRendering: false,
})

func (h *oauthHandler) renderTemplate(r *http.Request, w http.ResponseWriter, name string, m authModel) error {
	var err error

	ren := render.New(render.Options{
		Directory:  "templates/",
		Extensions: []string{".html"},
		Funcs: []template.FuncMap{{
			"HTTPErrors": func(err error) []errors.Http { _, errs := errors.HttpErrors(err); return errs },
		}},
		Delims:                    render.Delims{Left: "{{", Right: "}}"},
		Charset:                   "UTF-8",
		DisableCharset:            false,
		BinaryContentType:         "application/octet-stream",
		HTMLContentType:           "text/html",
		IsDevelopment:             true,
		DisableHTTPErrorRendering: false,
	})

	if err = ren.HTML(w, http.StatusOK, name, m); err != nil {
		new := errors.Annotatef(err, "failed to render template")
		h.logger.WithFields(logrus.Fields{
			"template": name,
			"model":    fmt.Sprintf("%T", m),
		}).Error(new.Error())
		errRenderer.HTML(w, http.StatusInternalServerError, "error", new)
		return err
	}
	return err
}

// ShowLogin serves GET /login requests
func (h *oauthHandler) ShowLogin(w http.ResponseWriter, r *http.Request) {
	a := activitypub.Self(pub.IRI(h.baseURL))

	m := login{title: "Login"}
	m.account = a

	h.renderTemplate(r, w, "login", m)
}

var errUnauthorized = errors.Unauthorizedf("Invalid username or password")

// ShowLogin handles POST /login requests
func (h *oauthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	a := activitypub.Self(pub.IRI(h.baseURL))

	pw := r.PostFormValue("pw")
	handle := r.PostFormValue("handle")
	client := r.PostFormValue("client")
	state := r.PostFormValue("state")

	h.logger.WithFields(logrus.Fields{
		"handle": handle,
		"pass":   pw,
		"client": client,
		"state":  state,
	}).Info("received")

	f := activitypub.FiltersNew()
	f.Name = activitypub.CompStrs{activitypub.CompStr{Str: handle}}
	f.IRI = activitypub.ActorsType.IRI(a)
	f.Type = []pub.ActivityVocabularyType{
		pub.PersonType,
	}
	actors, count, err := h.loader.LoadActors(f)
	if err != nil || count == 0 {
		errors.HandleError(errUnauthorized).ServeHTTP(w, r)
		return
	}

	if pwLoader, ok := h.loader.(st.PasswordChanger); ok {
		found := false
		for _, actor := range actors {
			_, err := checkPw(actor, []byte(pw), pwLoader)
			if err == nil {
				found = true
				break
			}
			if !found {
				if err == nil {
					err = errUnauthorized
				}
				h.logger.Error(err)
				errors.HandleError(err).ServeHTTP(w, r)
				return
			}
		}
	}
	config := oauth2.Config{
		ClientID: client,
		Endpoint: oauth2.Endpoint{
			AuthURL:  a.Endpoints.OauthAuthorizationEndpoint.GetLink().String(),
			TokenURL: a.Endpoints.OauthTokenEndpoint.GetLink().String(),
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

// HandleCallback serves /auth/callback request
func (h *oauthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {}

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
func (h *oauthHandler) ShowChangePw(w http.ResponseWriter, r *http.Request) {
	actor := h.loadActorFromOauth2Session(w, r)
	m := pwChange{
		title: "Change password",
	}
	if actor == nil {
		return
	}
	m.account = *actor

	h.renderTemplate(r, w, "password", m)
}

// HandleChangePw
func (h *oauthHandler) HandleChangePw(w http.ResponseWriter, r *http.Request) {
	actor := h.loadActorFromOauth2Session(w, r)
	if actor == nil {
		h.logger.Errorf("Unable to load actor from session")
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

	h.logger.WithFields(logrus.Fields{
		"handle": actor.PreferredUsername.String(),
		"pass":   pw,
	}).Info("received")

	if pwSetter, ok := h.loader.(st.PasswordChanger); ok {
		err := pwSetter.PasswordSet(actor, []byte(pw))
		if err != nil {
			h.logger.Errorf("Error when saving password: %s", err)
			errors.HandleError(errors.NotValidf("Unable to change password")).ServeHTTP(w, r)
			return
		}
		h.os.Storage.RemoveAuthorize(tok)
	}
}

func (h *oauthHandler) loadActorFromOauth2Session(w http.ResponseWriter, r *http.Request) *pub.Actor {
	notF := errors.NotFoundf("Not found")
	// TODO(marius): we land on this handler, coming from an email link containing a token identifying the Actor
	tok := r.URL.Query().Get("s")
	if len(tok) == 0 {
		h.logger.Errorf("Unable to load token from URL")
		errors.HandleError(notF).ServeHTTP(w, r)
		return nil
	}

	authSess, err := h.os.Storage.LoadAuthorize(tok)
	if err != nil {
		h.logger.Errorf("Error when loading authorize session: %s", err)
		errors.HandleError(notF).ServeHTTP(w, r)
		return nil
	}
	if authSess == nil {
		h.logger.Errorf("Invalid authorize session for tok %s", tok)
		errors.HandleError(notF).ServeHTTP(w, r)
		return nil
	}
	if authSess.ExpireAt().Sub(time.Now().UTC()) < 0 {
		h.logger.Errorf("Authorize token %s is expired %s", tok, authSess.ExpireAt().Format("2006-01-02 15:04:05"))
		errors.HandleError(notF).ServeHTTP(w, r)
		return nil
	}
	if authSess.UserData == nil {
		h.logger.Errorf("Invalid authorize session for tok %s, user-data is empty", tok)
		errors.HandleError(notF).ServeHTTP(w, r)
		return nil
	}

	actorIRI, ok := authSess.UserData.(string)
	if !ok {
		h.logger.Errorf("Invalid authorize session for tok %s, user-data is not an IRI: %v", tok, authSess.UserData)
		errors.HandleError(notF).ServeHTTP(w, r)
		return nil
	}
	actors, cnt, err := h.loader.LoadActors(pub.IRI(actorIRI))
	if err != nil {
		h.logger.Errorf("Error when loading actor from storage: %s", err)
		errors.HandleError(notF).ServeHTTP(w, r)
		return nil
	}
	if cnt > 1 {
		h.logger.Errorf("Ooops, too many actors in authorization session: %d, %v", cnt, actors)
		errors.HandleError(notF).ServeHTTP(w, r)
		return nil
	}
	var actor *pub.Actor
	pub.OnActor(actors[0], func(p *pub.Actor) error {
		actor = p
		return nil
	})

	return actor
}
