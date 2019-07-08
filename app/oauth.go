package app

import (
	"encoding/json"
	"github.com/go-ap/errors"
	"github.com/openshift/osin"
	"github.com/sirupsen/logrus"
	"net/http"
)

type account struct {}

func (a account) IsLogged() bool {
	return true
}

type oauthHandler struct {
	os *osin.Server
	account account
	logger logrus.FieldLogger
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
			if err := json.Unmarshal([]byte(who), &h.account); err == nil {
				ar.Authorized = h.account.IsLogged()
			} else {
				h.logger.Errorf("%s", err)
			}
		}
		s.FinishAccessRequest(resp, r, ar)
	}
	redirectOrOutput(resp, w, r, h)
}

func redirectOrOutput (rs *osin.Response, w http.ResponseWriter, r *http.Request, h *oauthHandler) {
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
			errors.HandleError(err).ServeHTTP(w, r)
			return
		}

		status := http.StatusUnauthorized
		if err := h.saveSession(w, r); err != nil {
			status = http.StatusFound
			h.logger.WithFields(logrus.Fields{
				"status": status,
				"url":    url,
			}).Error(err.Error())
		}

		http.Redirect(w, r, url, status)
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
