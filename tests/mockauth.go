package tests

import (
	"encoding/json"
	"fmt"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/oauth"
	"github.com/go-ap/fedbox/storage/boltdb"
	"github.com/openshift/osin"
	"github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func osinServer() (*osin.Server, error) {
	path := fmt.Sprintf("%s/host-oauth-%d.bdb", os.TempDir(), os.Getpid())
	l := logrus.New()
	return oauth.New(boltdb.NewOAuthStore(boltdb.Config{
		Path:       path,
		BucketName: host,
		LogFn:      func(f logrus.Fields, s string, p ...interface{}) { l.Errorf(s, p...) },
		ErrFn:      func(f logrus.Fields, s string, p ...interface{}) { l.Infof(s, p...) },
	}), l)
}
func osinAccess(s *osin.Server) (*osin.Response, error) {
	resp := s.NewResponse()
	defer resp.Close()

	v := url.Values{}
	v.Add("request_uri", url.QueryEscape("http://127.0.0.3/auth/local/callback"))
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
		v.Add("request_uri", url.QueryEscape("http://127.0.0.3/auth/local/callback"))
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
