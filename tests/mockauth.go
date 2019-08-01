package tests

import (
	"encoding/json"
	"fmt"
	"github.com/go-ap/errors"
	"github.com/openshift/osin"
	"github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"os"
	"strings"
)

var _oauthServer *osin.Server

type logger struct {
	l logrus.FieldLogger
}
func (l logger) Printf(s string, p ...interface{}) {
	l.l.Infof(s, p...)
}

func osinServer(store osin.Storage, l logrus.FieldLogger) (*osin.Server, error) {
	config := osin.ServerConfig{
		AuthorizationExpiration:   86400,
		AccessExpiration:          2678400,
		TokenType:                 "Bearer",
		AllowedAuthorizeTypes:     osin.AllowedAuthorizeType{osin.CODE},
		AllowedAccessTypes:        osin.AllowedAccessType{osin.AUTHORIZATION_CODE},
		ErrorStatusCode:           http.StatusForbidden,
		AllowClientSecretInParams: true,
		AllowGetAccessRequest:     false,
		RetainTokenAfterRefresh:   true,
		RedirectUriSeparator:      "\n",
		//RequirePKCEForPublicClients: true,
	}
	s := osin.NewServer(&config, store)
	s.Logger = &logger{ l: l }

	return s, nil
}

func osinAccess(s *osin.Server) (*osin.Response, error) {
	resp := s.NewResponse()
	defer resp.Close()

	v := url.Values{}
	v.Add("request_uri", url.QueryEscape(authCallbackURL))
	v.Add("client_id", os.Getenv("OAUTH2_KEY"))
	v.Add("response_type", "code")
	dummyAuthReq, _ := http.NewRequest(http.MethodGet, "/oauth2/authorize", nil)
	dummyAuthReq.URL.RawQuery = v.Encode()
	ar := s.HandleAuthorizeRequest(resp, dummyAuthReq)
	if ar == nil {
		return resp, errors.BadRequestf("invalid authorize req")
	}
	b, _ := json.Marshal(defaultTestAccount)
	ar.UserData = b
	ar.Authorized = true
	s.FinishAuthorizeRequest(resp, dummyAuthReq, ar)

	return resp, nil
}

func addOAuth2Auth(r *http.Request, a *testAccount, s *osin.Server) error {
	resp, err := osinAccess(s)
	if err != nil {
		return err
	}
	if d := resp.Output["code"]; d != nil {
		cod, ok := d.(string)
		if !ok {
			return errors.BadRequestf("unable to finish authorize req, bad response code %s", cod)
		}
		resp := s.NewResponse()
		defer resp.Close()
		key := os.Getenv("OAUTH2_KEY")
		sec := os.Getenv("OAUTH2_SECRET")
		v := url.Values{}
		v.Add("request_uri", url.QueryEscape(authCallbackURL))
		v.Add("client_id", key)
		v.Add("client_secret", sec)
		v.Add("access_type", "online")
		v.Add("grant_type", "authorization_code")
		v.Add("state", "state")
		v.Add("code", cod)
		dummyAccessReq, _ := http.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(v.Encode()))
		dummyAccessReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		ar := s.HandleAccessRequest(resp, dummyAccessReq)
		if ar == nil {
			return errors.BadRequestf("invalid access req")
		}
		ar.Authorized = true
		b, _ := json.Marshal(a)
		ar.UserData = b
		ar.ForceAccessData = ar.AccessData
		s.FinishAccessRequest(resp, dummyAccessReq, ar)

		if cod := resp.Output["access_token"]; d != nil {
			tok, okK := cod.(string)
			typ, okP := resp.Output["token_type"].(string)
			if okK && okP {
				r.Header.Set("Authorization", fmt.Sprintf("%s %s", typ, tok))
				return nil
			}
		}
		return errors.BadRequestf("unable to finish access req, bad response token %s", cod)
	}
	return errors.Newf("unknown :D")
}
