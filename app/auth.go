package app

import (
	"crypto"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"github.com/go-ap/activitypub/client"
	cl "github.com/go-ap/activitypub/client"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/activitypub"
	st "github.com/go-ap/storage"
	"github.com/openshift/osin"
	"github.com/sirupsen/logrus"
	"github.com/spacemonkeygo/httpsig"
	"net/http"
	"strings"
)

var oss *osin.Server

type keyLoader struct {
	baseIRI string
	logFn   func(string, ...interface{})
	realm   string
	acc     activitypub.Person
	l       st.ActorLoader
	c       client.Client
}

func loadFederatedActor(c client.Client, id as.IRI) (activitypub.Person, error) {
	it, err := c.LoadIRI(id)
	if err != nil {
		return activitypub.AnonymousActor, err
	}
	if acct, ok := it.(*activitypub.Person); ok {
		return *acct, nil
	}
	if acct, ok := it.(activitypub.Person); ok {
		return acct, nil
	}
	return activitypub.AnonymousActor, nil
}

func validateLocalIRI(i as.IRI) error {
	if strings.Contains(i.String(), Config.BaseURL) {
		return nil
	}
	return errors.Newf("%s is not a local IRI", i)
}

func (k *keyLoader) GetKey(id string) interface{} {
	var err error

	iri := as.IRI(id)
	u, err := iri.URL()
	if err != nil {
		return err
	}
	if u.Fragment != "main-key" {
		// invalid generated public key id
		k.logFn("missing key")
		return nil
	}

	if err := validateLocalIRI(iri); err == nil {
		actors, cnt, err := k.l.LoadActors(&activitypub.Filters{IRI: iri})
		if err != nil || cnt == 0 {
			k.logFn("unable to find local account matching key id %s", iri)
			return nil
		}
		actor := actors.First()
		if acct, ok := actor.(*activitypub.Person); ok {
			k.acc = *acct
		}
		if acct, ok := actor.(activitypub.Person); ok {
			k.acc = acct
		}
	} else {
		// @todo(queue_support): this needs to be moved to using queues
		k.acc, err = loadFederatedActor(k.c, iri)
		if err != nil {
			k.logFn("unable to load federated account matching key id %s", iri)
			return nil
		}
	}

	obj, err := activitypub.ToPerson(k.acc)
	if err != nil {
		k.logFn("unable to load actor %s", err)
		return nil
	}
	var pub crypto.PublicKey
	rawPem := obj.PublicKey.PublicKeyPem
	block, _ := pem.Decode([]byte(rawPem))
	if block == nil {
		k.logFn("failed to parse PEM block containing the public key")
		return nil
	}
	pub, err = x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		k.logFn("x509 error %s", err)
		return nil
	}
	return pub
}

type oauthLoader struct {
	logFn func(string, ...interface{})
	acc   activitypub.Person
	s     *osin.Server
}

func (k *oauthLoader) Verify(r *http.Request) (error, string) {
	bearer := osin.CheckBearerAuth(r)
	dat, err := k.s.Storage.LoadAccess(bearer.Code)
	if err != nil {
		return err, ""
	}
	if b, ok := dat.UserData.(json.RawMessage); ok {
		if err := json.Unmarshal(b, &k.acc); err != nil {
			return err, ""
		}
	} else {
		return errors.Unauthorizedf("unable to load from bearer"), ""
	}
	return nil, ""
}

func httpSignatureVerifier(getter *keyLoader) (*httpsig.Verifier, string) {
	v := httpsig.NewVerifier(getter)
	v.SetRequiredHeaders([]string{"(request-target)", "host", "date"})

	var challengeParams []string
	if getter.realm != "" {
		challengeParams = append(challengeParams, fmt.Sprintf("realm=%q", getter.realm))
	}
	if headers := v.RequiredHeaders(); len(headers) > 0 {
		challengeParams = append(challengeParams, fmt.Sprintf("headers=%q", strings.Join(headers, " ")))
	}

	challenge := "Signature"
	if len(challengeParams) > 0 {
		challenge += fmt.Sprintf(" %s", strings.Join(challengeParams, ", "))
	}
	return v, challenge
}

func LoadActorFromAuthHeader(r *http.Request, l logrus.FieldLogger) (as.Actor, error) {
	client := cl.NewClient()
	acct := activitypub.AnonymousActor
	if auth := r.Header.Get("Authorization"); auth != "" {
		var err error
		var challenge string
		method := "none"
		if strings.Contains(auth, "Bearer") {
			// check OAuth2 bearer if present
			method = "oauth2"
			// TODO(marius): move this to a better place but outside the handler
			v := oauthLoader{acc: acct, s: oss}
			v.logFn = l.WithFields(logrus.Fields{"from": method}).Debugf
			err, challenge = v.Verify(r)
			acct = v.acc
		}
		if strings.Contains(auth, "Signature") {
			if loader, ok := actorLoader(r.Context()); ok {
				// only verify http-signature if present
				getter := keyLoader{acc: acct, l: loader, realm: r.URL.Host, c: client}
				method = "httpSig"
				getter.logFn = l.WithFields(logrus.Fields{"from": method}).Debugf

				var v *httpsig.Verifier
				v, challenge = httpSignatureVerifier(&getter)
				err = v.Verify(r)
				acct = getter.acc
			}
		}
		if err != nil {
			// TODO(marius): fix this challenge passing
			err = errors.NewUnauthorized(err, "").Challenge(r.URL.Path)
			l.WithFields(logrus.Fields{
				"id":        acct.GetID(),
				"auth":      r.Header.Get("Authorization"),
				"req":       fmt.Sprintf("%s:%s", r.Method, r.URL.RequestURI()),
				"err":       err,
				"challenge": challenge,
			}).Warn("invalid HTTP Authorization")
			// TODO(marius): here we need to implement some outside logic, as to we want to allow non-signed
			//   requests on some urls, but not on others - probably another handler to check for Anonymous
			//   would suffice.
			return acct, err
		} else {
			// TODO(marius): Add actor's host to the logging
			l.WithFields(logrus.Fields{
				"auth": method,
				"id":   acct.GetID(),
			}).Debug("loaded account from Authorization header")
		}
	}
	return acct, nil
}
