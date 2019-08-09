package app

import (
	"encoding/json"
	"fmt"
	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/oauth"
	"github.com/openshift/osin"
	"github.com/sirupsen/logrus"
	"github.com/unrolled/render"
	"golang.org/x/oauth2"
	"html/template"
	"net/http"
	"os"
	"time"
)

type account struct{}

func (a account) IsLogged() bool {
	return true
}

type oauthHandler struct {
	os      *osin.Server
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
			b, _ := json.Marshal(h.account)
			ar.UserData = b
		}
		s.FinishAuthorizeRequest(resp, r, ar)
	}
	redirectOrOutput(resp, w, r, h)
}

func (h *oauthHandler) Token(w http.ResponseWriter, r *http.Request) {
	s := h.os
	resp := s.NewResponse()
	defer resp.Close()

	if ar := s.HandleAccessRequest(resp, r); ar != nil {
		if who, ok := ar.UserData.(json.RawMessage); ok {
			if err := json.Unmarshal(who, &h.account); err != nil {
				err := errors.Annotatef(err, "Unable to unmarshal user-data to local actor")
				h.logger.Errorf("%s", err)
				errors.HandleError(err).ServeHTTP(w, r)
				return
			}
			ar.Authorized = h.account.IsLogged()
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
		err := annotatedRsError(rs.StatusCode, rs.InternalError, "Error processing OAuth2 request")
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

		if err := h.saveSession(w, r); err != nil {
			st := http.StatusInternalServerError
			w.WriteHeader(st)
			h.logger.WithFields(logrus.Fields{
				"status": st,
				"url":    url,
			}).Error(err.Error())
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
		if err := h.saveSession(w, r); err != nil {
			errors.HandleError(err).ServeHTTP(w, r)
			return
		}
	}
}

func (h *oauthHandler) saveSession(w http.ResponseWriter, r *http.Request) error {
	//if h.sstor == nil {
	//	err := errors.New("missing session store, unable to save session")
	//	h.logger.Errorf("%s", err)
	//	return err
	//}
	//s, err := h.sstor.Get(r, sessionName)
	//if err != nil {
	//	h.logger.Errorf("%s", err)
	//	return errors.Errorf("failed to load session before redirect: %s", err)
	//}
	//if err := h.sstor.Save(r, w, s); err != nil {
	//	err := errors.Errorf("failed to save session before redirect: %s", err)
	//	h.logger.Errorf("%s", err)
	//	return err
	//}
	return nil
}

type login struct {
	title   string
	account auth.Person
}

func (l login) Title() string {
	return l.title
}

func (l login) Account() auth.Person {
	return l.account
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

func (h *oauthHandler) RenderTemplate(r *http.Request, w http.ResponseWriter, name string, m authModel) error {
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
	if err = h.saveSession(w, r); err != nil {
		h.logger.WithFields(logrus.Fields{
			"template": name,
			"model":    fmt.Sprintf("%#v", m),
		}).Error(err.Error())
	}
	return err
}

// ShowLogin serves GET /login requests
func (h *oauthHandler) ShowLogin(w http.ResponseWriter, r *http.Request) {
	a := activitypub.Self("http://fedbox.git")

	m := login{title: "Login"}
	m.account = a

	h.RenderTemplate(r, w, "login", m)
}

// ShowLogin handles POST /login requests
func (h *oauthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	a := activitypub.Self("http://fedbox.git")
	st, err := oauth.New(h.os.Storage, h.logger)
	if err != nil {
		new := errors.Annotatef(err, "failed to load default client")
		h.logger.WithFields(logrus.Fields{}).Error(new.Error())
		errRenderer.HTML(w, http.StatusInternalServerError, "error", new)
		return
	}
	cl, err := st.Storage.GetClient(os.Getenv("OAUTH_CLIENT"))
	if err != nil {
		new := errors.Annotatef(err, "failed to load default client")
		h.logger.WithFields(logrus.Fields{}).Error(new.Error())
		errRenderer.HTML(w, http.StatusInternalServerError, "error", new)
		return
	}
	pw := r.PostFormValue("pw")
	handle := r.PostFormValue("handle")

	h.logger.WithFields(logrus.Fields{
		"handle": handle,
		"pass":   pw,
	}).Info("received")

	config := oauth2.Config{
		ClientID:     cl.GetId(),
		ClientSecret: cl.GetSecret(),
		Endpoint: oauth2.Endpoint{
			AuthURL:  a.Endpoints.OauthAuthorizationEndpoint.GetLink().String(),
			TokenURL: a.Endpoints.OauthTokenEndpoint.GetLink().String(),
		},
	}
	http.Redirect(w, r, config.AuthCodeURL("state", oauth2.AccessTypeOnline), http.StatusPermanentRedirect)
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

// HandleCallback serves /auth/{provider}/callback request
func (h *oauthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	provider := "fedbox"

	providerErr := q["error"]
	if providerErr != nil {
		errDescriptions := q["error_description"]
		err := errors.Errorf("Error for provider %q: %s\n", provider, errDescriptions)
		errRenderer.HTML(w, http.StatusInternalServerError, "error", err)
		return
	}
	code := q.Get("code")
	state := q.Get("state")
	if len(code) == 0 {
		errRenderer.HTML(w, http.StatusInternalServerError, "error", errors.Forbiddenf("%s error: Empty authentication token", provider))
		return
	}

	a := activitypub.Self("http://fedbox.git")
	st, err := oauth.New(h.os.Storage, h.logger)
	if err != nil {
		new := errors.Annotatef(err, "failed to load default client")
		h.logger.WithFields(logrus.Fields{}).Error(new.Error())
		errRenderer.HTML(w, http.StatusInternalServerError, "error", new)
		return
	}
	cl, err := st.Storage.GetClient(os.Getenv("OAUTH_CLIENT"))
	if err != nil {
		new := errors.Annotatef(err, "failed to load default client")
		h.logger.WithFields(logrus.Fields{}).Error(new.Error())
		errRenderer.HTML(w, http.StatusInternalServerError, "error", new)
		return
	}

	conf := oauth2.Config{
		ClientID:     cl.GetId(),
		ClientSecret: cl.GetSecret(),
		Endpoint: oauth2.Endpoint{
			AuthURL:  a.Endpoints.OauthAuthorizationEndpoint.GetLink().String(),
			TokenURL: a.Endpoints.OauthTokenEndpoint.GetLink().String(),
		},
	}
	tok, err := conf.Exchange(r.Context(), code)
	if err != nil {
		h.logger.Errorf("%s", err)
		errRenderer.HTML(w, http.StatusInternalServerError, "error", err)
		//h.HandleErrors(w, r, err)
		return
	}

	oauth := OAuth{
		State:        state,
		Code:         code,
		Provider:     provider,
		Token:        tok.AccessToken,
		TokenType:    tok.TokenType,
		RefreshToken: tok.RefreshToken,
		Expiry:       tok.Expiry,
	}
	h.logger.WithField("oauth", oauth).Infof("Success")
	http.Redirect(w, r, "/", http.StatusFound)
}
