package activitypub

import (
	"fmt"
	"net/url"
	"path"

	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/handlers"
	"github.com/pborman/uuid"
)

const developer = pub.IRI("https://github.com/mariusor")

func Self(baseURL pub.IRI) pub.Service {
	url, _ := baseURL.URL()
	oauth := *url
	oauth.Path = path.Join(oauth.Path, "oauth/")
	s := pub.Service{
		ID:           baseURL,
		Type:         pub.ServiceType,
		Name:         pub.NaturalLanguageValues{{Ref: pub.NilLangRef, Value: pub.Content("self")}},
		AttributedTo: developer,
		Audience:     pub.ItemCollection{pub.PublicNS},
		Content:      nil, //pub.NaturalLanguageValues{{Ref: pub.NilLangRef, Value: ""}},
		Summary:      pub.NaturalLanguageValues{{Ref: pub.NilLangRef, Value: pub.Content("Generic ActivityPub service")}},
		Tag:          nil,
		URL:          baseURL,
		Endpoints: &pub.Endpoints{
			OauthAuthorizationEndpoint: pub.IRI(fmt.Sprintf("%s/authorize", oauth.String())),
			OauthTokenEndpoint:         pub.IRI(fmt.Sprintf("%s/token", oauth.String())),
		},
	}

	s.Inbox = handlers.Inbox.IRI(s)
	s.Outbox = handlers.Outbox.IRI(s)
	s.Streams = pub.ItemCollection{ActorsType.IRI(s), ActivitiesType.IRI(s), ObjectsType.IRI(s)}
	return s
}

func DefaultServiceIRI(baseURL string) pub.IRI {
	u, _ := url.Parse(baseURL)
	// TODO(marius): I don't like adding the / folder to something like http://fedbox.git
	// I need to find an
	if u.Path == "" {
		u.Path = "/"
	}
	return pub.IRI(u.String())
}

// GenerateID generates an unique identifier for the it ActivityPub Object.
func GenerateID(it pub.Item, partOf pub.IRI, by pub.Item) (pub.ID, error) {
	uuid := uuid.New()
	id := partOf.GetLink().AddPath(uuid)
	if pub.ActivityTypes.Contains(it.GetType()) {
		err := pub.OnActivity(it, func(a *pub.Activity) error {
			rec := append(a.To, append(a.CC, append(a.Bto, a.BCC...)...)...)
			if !rec.Contains(pub.PublicNS) {
				if pub.IsNil(by) {
					by = a.Actor
				}
				if !pub.IsNil(by) {
					// if it's not a public activity, save it to it's actor outbox instead of global activities collection
					outbox := handlers.Outbox.IRI(by)
					id = pub.ID(fmt.Sprintf("%s/%s", outbox, uuid))
				}
			}
			a.ID = id
			return nil
		})
		return id, err
	}
	if it.IsLink() {
		return id, pub.OnLink(it, func(l *pub.Link) error {
			l.ID = id
			return nil
		})
	}
	return id, pub.OnObject(it, func(o *pub.Object) error {
		o.ID = id
		return nil
	})
	return id, nil
}
