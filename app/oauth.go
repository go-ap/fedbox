package app

import (
	"encoding/json"
	"fmt"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/cmd"
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
	actor    *auth.Person
}

func (a account) IsLogged() bool {
	return a.actor != nil && a.actor.PreferredUsername.First().Value == a.username
}

type oauthHandler struct {
	baseURL string
	os      *osin.Server
	loader  storage.ActorLoader
	account account
	logger  logrus.FieldLogger
}

func (h *oauthHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	s := h.os

	resp := s.NewResponse()
	defer resp.Close()

	if ar := s.HandleAuthorizeRequest(resp, r); ar != nil {
		if h.account.IsLogged() {
			ar.Authorized = true
			ar.UserData = h.account.actor.GetLink()
		} else {
			m := login{title: "Login"}
			m.account = auth.AnonymousActor
			m.client = ar.Client.GetId()
			m.state = ar.State

			h.renderTemplate(r, w, "login", m)
			return
		}
		s.FinishAuthorizeRequest(resp, r, ar)
	}
	redirectOrOutput(resp, w, r, h)
}

func checkPw(it as.Item, pw []byte, pwLoader cmd.PasswordChanger) (account, error) {
	acc := account{
		username: "anonymous",
		actor:    &auth.AnonymousActor,
	}
	err := auth.OnPerson(it, func(p *auth.Person) error {
		err := pwLoader.PasswordCheck(p, []byte(pw))
		if err != nil {
			// TODO(marius): log the received error
			return errors.Unauthorizedf("Invalid username or password")
		}
		acc = account{
			username: p.PreferredUsername.String(),
			actor:    p,
		}
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

	if ar := s.HandleAccessRequest(resp, r); ar != nil {
		actorFilters := activitypub.Filters{}
		switch ar.Type {
		case osin.PASSWORD:
			actorFilters.IRI = as.IRI(fmt.Sprintf("%s/actors", h.baseURL))
			actorFilters.Name = []string{ar.Username}
		case osin.AUTHORIZATION_CODE:
			if iri, ok := ar.UserData.(string); ok {
				actorFilters.IRI = as.IRI(iri)
			}
		}
		actors, _, err := h.loader.LoadActors(actorFilters)
		if err != nil {
			h.logger.Error(errUnauthorized)
			errors.HandleError(errUnauthorized).ServeHTTP(w, r)
			return
		}
		if ar.Type == osin.PASSWORD {
			if pwLoader, ok := h.loader.(cmd.PasswordChanger); ok {
				found := false
				for _, actor := range actors {
					h.account, err = checkPw(actor, []byte(ar.Password), pwLoader)
					if err == nil {
						ar.Authorized = h.account.IsLogged()
						ar.UserData = h.account.actor.GetLink()
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
			auth.OnPerson(actors.First(), func(p *auth.Person) error {
				h.account = account{
					username: p.PreferredUsername.String(),
					actor:    p,
				}
				ar.Authorized = h.account.IsLogged()
				ar.UserData = h.account.actor.GetLink()
				return nil
			})
		}
		s.FinishAccessRequest(resp, r, ar)
	}
	redirectOrOutput(resp, w, r, h)
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

func redirectOrOutput(rs *osin.Response, w http.ResponseWriter, r *http.Request, h *oauthHandler) {
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
	account auth.Person
	state   string
	client  string
}

func (l login) Title() string {
	return l.title
}

func (l login) Account() auth.Person {
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
	Account() auth.Person
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
	a := activitypub.Self(as.IRI(h.baseURL))

	m := login{title: "Login"}
	m.account = a

	h.renderTemplate(r, w, "login", m)
}

var errUnauthorized = errors.Unauthorizedf("Invalid username or password")

// ShowLogin handles POST /login requests
func (h *oauthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	a := activitypub.Self(as.IRI(h.baseURL))

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

	filter := activitypub.Filters{
		Name: []string{handle},
		IRI:  as.IRI(fmt.Sprintf("%s/actors", a.GetLink())),
		Type: []as.ActivityVocabularyType{
			as.PersonType,
		},
	}
	actors, count, err := h.loader.LoadActors(filter)
	if err != nil || count == 0 {
		errors.HandleError(errUnauthorized).ServeHTTP(w, r)
		return
	}

	if pwLoader, ok := h.loader.(cmd.PasswordChanger); ok {
		found := false
		for _, actor := range actors {
			h.account, err = checkPw(actor, []byte(pw), pwLoader)
			if err == nil {
				found = true
				break
			}
			if !found {
				h.logger.Error(errUnauthorized)
				errors.HandleError(errUnauthorized).ServeHTTP(w, r)
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
