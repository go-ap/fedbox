// +build integration

package tests

import (
	"fmt"
	"net/http"

	"github.com/openshift/osin"
	"github.com/sirupsen/logrus"
)

func osinServer(store osin.Storage, l logrus.FieldLogger) *osin.Server {
	config := osin.NewServerConfig()
	config.AuthorizationExpiration = 86400
	config.AccessExpiration = 2678400
	config.ErrorStatusCode = http.StatusForbidden
	config.AllowClientSecretInParams = true
	config.RetainTokenAfterRefresh = true
	config.RedirectUriSeparator = "\n"
	//config.RequirePKCEForPublicClients = true
	s := osin.NewServer(config, store)
	s.Logger = l
	return s
}

func addOAuth2Auth(r *http.Request, a *testAccount) error {
	r.Header.Add("Authorization", fmt.Sprintf("Bearer %s", a.AuthToken))
	return nil
}
