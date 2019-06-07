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
	"net/url"
	"path"
	"strings"
)

var oss *osin.Server

type keyLoader struct {
	logFn func(string, ...interface{})
	realm string
	acc   as.Actor
	l     st.ActorLoader
	c     client.Client
}

func loadFederatedActor(c client.Client, id as.IRI) (as.Actor, error) {
	it, err := c.LoadIRI(id)
	if err != nil {
		return as.Person{}, err
	}
	if acct, ok := it.(*as.Person); ok {
		return acct, nil
	}
	return as.Person{}, nil
}

func (k *keyLoader) GetKey(id string) interface{} {
	var err error

	u, err := url.Parse(id)
	if err != nil {
		return err
	}
	if u.Fragment != "main-key" {
		// invalid generated public key id
		k.logFn("missing key")
		return nil
	}

	if err := validateLocalIRI(as.IRI(id)); err == nil {
		hash := path.Base(u.Path)
		k.acc, _, err = k.l.LoadActors(&activitypub.Filters{Key: []activitypub.Hash{activitypub.Hash(hash)}})
		if err != nil {
			k.logFn("unable to find local account matching key id %s", id)
			return nil
		}
	} else {
		// @todo(queue_support): this needs to be moved to using queues
		k.acc, err = loadFederatedActor(k.c, as.IRI(id))
		if err != nil {
			k.logFn("unable to load federated account matching key id %s", id)
			return nil
		}
	}

	obj, err := activitypub.ToPerson(k.acc)
	if err != nil {
		k.logFn("unable to load actor %s", err)
		return nil
	}
	var pub crypto.PublicKey
	pemmed := obj.PublicKey.PublicKeyPem
	block, _ := pem.Decode([]byte(pemmed))
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
	acc   as.Actor
	s     *osin.Server
}

func (k *oauthLoader) Verify(r *http.Request) (error, string) {
	bearer := osin.CheckBearerAuth(r)
	dat, err := k.s.Storage.LoadAccess(bearer.Code)
	if err != nil {
		return err, ""
	}
	if b, ok := dat.UserData.(json.RawMessage); ok {
		if err := json.Unmarshal([]byte(b), &k.acc); err != nil {
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
	var acct as.Actor
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
			if loader, ok := loader(r.Context()); ok {
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
			err = errors.NewUnauthorized(err, "").Challenge(challenge)
			l.WithFields(logrus.Fields{
				"id":   acct.GetID(),
				"auth": r.Header.Get("Authorization"),
				"req":  fmt.Sprintf("%s:%s", r.Method, r.URL.RequestURI()),
				"err":  err,
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
