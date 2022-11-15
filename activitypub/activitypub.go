package activitypub

import (
	"fmt"
	"net/url"
	"path"

	vocab "github.com/go-ap/activitypub"
	"github.com/pborman/uuid"
)

const developer = vocab.IRI("https://github.com/mariusor")

func Self(baseURL vocab.IRI) vocab.Service {
	url, _ := baseURL.URL()
	oauth := *url
	oauth.Path = path.Join(oauth.Path, "oauth/")
	s := vocab.Service{
		ID:           baseURL,
		Type:         vocab.ServiceType,
		Name:         vocab.NaturalLanguageValues{{Ref: vocab.NilLangRef, Value: vocab.Content("self")}},
		AttributedTo: developer,
		Audience:     vocab.ItemCollection{vocab.PublicNS},
		Content:      nil, //vocab.NaturalLanguageValues{{Ref: vocab.NilLangRef, Value: ""}},
		Summary:      vocab.NaturalLanguageValues{{Ref: vocab.NilLangRef, Value: vocab.Content("Generic ActivityPub service")}},
		Tag:          nil,
		URL:          baseURL,
		Endpoints: &vocab.Endpoints{
			OauthAuthorizationEndpoint: vocab.IRI(fmt.Sprintf("%s/authorize", oauth.String())),
			OauthTokenEndpoint:         vocab.IRI(fmt.Sprintf("%s/token", oauth.String())),
		},
	}

	s.Inbox = vocab.Inbox.IRI(s)
	s.Outbox = vocab.Outbox.IRI(s)
	s.Streams = vocab.ItemCollection{ActorsType.IRI(s), ActivitiesType.IRI(s), ObjectsType.IRI(s)}
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

// GenerateID generates an unique identifier for the it ActivityPub Object.
func GenerateID(it vocab.Item, partOf vocab.IRI, by vocab.Item) (vocab.ID, error) {
	uuid := uuid.New()
	id := partOf.GetLink().AddPath(uuid)
	typ := it.GetType()
	if vocab.ActivityTypes.Contains(typ) || vocab.IntransitiveActivityTypes.Contains(typ) {
		err := vocab.OnActivity(it, func(a *vocab.Activity) error {
			rec := append(a.To, append(a.CC, append(a.Bto, a.BCC...)...)...)
			if !rec.Contains(vocab.PublicNS) {
				if vocab.IsNil(by) {
					by = a.Actor
				}
				if !vocab.IsNil(by) {
					// if it's not a public activity, save it to it's actor outbox instead of global activities collection
					outbox := vocab.Outbox.IRI(by)
					id = vocab.ID(fmt.Sprintf("%s/%s", outbox, uuid))
				}
			}
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
