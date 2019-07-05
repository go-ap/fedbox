package oauth

import (
	"github.com/openshift/osin"
	"github.com/sirupsen/logrus"
	"net/http"
)

type logger struct {
	l logrus.FieldLogger
}

func (l logger) Printf(format string, v ...interface{}) {
	l.l.Debugf(format, v...)
}

func New(store osin.Storage, l logrus.FieldLogger) (*osin.Server, error) {
	config := osin.ServerConfig{
		AuthorizationExpiration:   86400,
		AccessExpiration:          2678400,
		TokenType:                 "Bearer",
		AllowedAuthorizeTypes:     osin.AllowedAuthorizeType{osin.CODE},
		AllowedAccessTypes:        osin.AllowedAccessType{osin.AUTHORIZATION_CODE},
		ErrorStatusCode:           http.StatusForbidden,
		AllowClientSecretInParams: false,
		AllowGetAccessRequest:     false,
		RetainTokenAfterRefresh:   true,
		RedirectUriSeparator:      "\n",
		//RequirePKCEForPublicClients: true,
	}
	s := osin.NewServer(&config, store)
	s.Logger = logger{l: l}
	return s, nil
}
