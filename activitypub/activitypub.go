package ap

import (
	"fmt"
	"net/url"
	"path"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	"github.com/go-ap/filters"
	"github.com/go-ap/processing"
)

const (
	developerURL = vocab.IRI("https://github.com/mariusor")
	ProjectURL   = vocab.IRI("https://github.com/go-ap/fedbox")
)

func Self(baseURL vocab.IRI, name string) vocab.Service {
	u, _ := baseURL.URL()
	oauth := *u
	oauth.Path = path.Join(oauth.Path, "oauth/")
	s := vocab.Service{
		ID:                baseURL,
		Type:              vocab.ServiceType,
		Name:              vocab.NaturalLanguageValuesNew(vocab.DefaultLangRef(name)),
		PreferredUsername: vocab.NaturalLanguageValuesNew(vocab.DefaultLangRef("self")),
		Context:           ProjectURL,
		AttributedTo:      developerURL,
		Audience:          vocab.ItemCollection{vocab.PublicNS},
		Content:           nil, //vocab.NaturalLanguageValues{{Ref: vocab.NilLangRef, Value: ""}},
		Summary:           vocab.NaturalLanguageValuesNew(vocab.DefaultLangRef("Generic ActivityPub service")),
		Tag:               nil,
		URL:               baseURL,
		Endpoints: &vocab.Endpoints{
			OauthAuthorizationEndpoint: vocab.IRI(fmt.Sprintf("%s/authorize", oauth.String())),
			OauthTokenEndpoint:         vocab.IRI(fmt.Sprintf("%s/token", oauth.String())),
		},
	}

	s.Inbox = vocab.Inbox.IRI(s)
	s.Outbox = vocab.Outbox.IRI(s)
	s.Streams = vocab.ItemCollection{
		filters.ActorsType.IRI(s),
		filters.ActivitiesType.IRI(s),
		filters.ObjectsType.IRI(s),
	}
	return s
}

func DefaultServiceIRI(baseURL string) vocab.IRI {
	u, _ := url.Parse(baseURL)
	// TODO(marius): I don't like adding the / folder to something like http://fedbox.git
	if u.Path == "" {
		u.Path = "/"
	}
	return vocab.IRI(u.String())
}

func LoadActor(st processing.ReadStore, iri vocab.IRI, ff ...filters.Check) (vocab.Actor, error) {
	var act vocab.Actor

	selfCol, err := st.Load(iri, ff...)
	if err != nil {
		return act, errors.Annotatef(err, "invalid service IRI %s", iri)
	}

	err = vocab.OnActor(selfCol, func(actor *vocab.Actor) error {
		act = *actor
		return nil
	})
	return act, err
}
