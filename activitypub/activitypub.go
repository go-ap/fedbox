package activitypub

import (
	"fmt"
	"net/url"
	"path"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	"github.com/go-ap/filters"
	"github.com/go-ap/processing"
	"github.com/pborman/uuid"
)

const (
	developerURL = vocab.IRI("https://github.com/mariusor")
	ProjectURL   = vocab.IRI("https://github.com/go-ap/fedbox")
)

func Self(baseURL vocab.IRI) vocab.Service {
	u, _ := baseURL.URL()
	oauth := *u
	oauth.Path = path.Join(oauth.Path, "oauth/")
	s := vocab.Service{
		ID:           baseURL,
		Type:         vocab.ServiceType,
		Name:         vocab.NaturalLanguageValuesNew(vocab.DefaultLangRef("self")),
		Context:      ProjectURL,
		AttributedTo: developerURL,
		Audience:     vocab.ItemCollection{vocab.PublicNS},
		Content:      nil, //vocab.NaturalLanguageValues{{Ref: vocab.NilLangRef, Value: ""}},
		Summary:      vocab.NaturalLanguageValuesNew(vocab.DefaultLangRef("Generic ActivityPub service")),
		Tag:          nil,
		URL:          baseURL,
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

// GenerateID generates a unique identifier for the 'it' [vocab.Item].
func GenerateID(it vocab.Item, partOf vocab.IRI, by vocab.Item) (vocab.ID, error) {
	uid := uuid.New()
	id := partOf.GetLink().AddPath(uid)
	typ := it.GetType()
	if vocab.ActivityTypes.Contains(typ) || vocab.IntransitiveActivityTypes.Contains(typ) {
		err := vocab.OnIntransitiveActivity(it, func(a *vocab.IntransitiveActivity) error {
			if rec := a.Recipients(); rec.Contains(vocab.PublicNS) {
				return nil
			}
			if vocab.IsNil(by) {
				by = a.Actor
			}
			if !vocab.IsNil(by) {
				// if "it" is not a public activity, save it to its actor Outbox instead of the global activities collection
				outbox := vocab.Outbox.IRI(by)
				id = vocab.ID(fmt.Sprintf("%s/%s", outbox, uid))
			}
			return nil
		})
		if err != nil {
			return id, err
		}
		err = vocab.OnObject(it, func(a *vocab.Object) error {
			a.ID = id
			return nil
		})
		return id, err
	}
	if it.IsLink() {
		return id, vocab.OnLink(it, func(l *vocab.Link) error {
			l.ID = id
			return nil
		})
	}
	return id, vocab.OnObject(it, func(o *vocab.Object) error {
		o.ID = id
		return nil
	})
	return id, nil
}
