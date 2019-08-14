package tests

import (
	"fmt"
	"github.com/openshift/osin"
	"github.com/sirupsen/logrus"
	"net/http"
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

func addOAuth2Auth(r *http.Request, a *testAccount) error {
	r.Header.Add("Authorization", fmt.Sprintf("Bearer %s", a.authToken))
	return nil
}
