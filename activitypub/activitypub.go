package activitypub

import (
	"fmt"
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/handlers"
	"github.com/pborman/uuid"
	"net/url"
	"path"
	"strings"
)

const developer = pub.IRI("https://github.com/mariusor")

func Self(baseURL pub.IRI) pub.Service {
	url, _ := baseURL.URL()
	inbox := *url
	inbox.Path = path.Join(inbox.Path, string(handlers.Inbox))

	oauth := *url
	oauth.Path = path.Join(oauth.Path, "oauth/")
	return pub.Service{
		ID:           pub.ID(url.String()),
		Type:         pub.ServiceType,
		Name:         pub.NaturalLanguageValues{{Ref: pub.NilLangRef, Value: pub.Content("self")}},
		AttributedTo: developer,
		Audience:     pub.ItemCollection{pub.PublicNS},
		Content:      nil, //pub.NaturalLanguageValues{{Ref: pub.NilLangRef, Value: ""}},
		Icon:         nil,
		Image:        nil,
		Location:     nil,
		Summary:      pub.NaturalLanguageValues{{Ref: pub.NilLangRef, Value: pub.Content("Generic ActivityPub service")}},
		Tag:          nil,
		URL:          baseURL,
		Inbox:        pub.IRI(inbox.String()),
		Endpoints: &pub.Endpoints{
			OauthAuthorizationEndpoint: pub.IRI(fmt.Sprintf("%s/authorize", oauth.String())),
			OauthTokenEndpoint:         pub.IRI(fmt.Sprintf("%s/token", oauth.String())),
		},
	}
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
func GenerateID(it pub.Item, partOf string, by pub.Item) (pub.ID, error) {
	uuid := uuid.New()
	id := pub.ID(fmt.Sprintf("%s/%s", strings.ToLower(partOf), uuid))
	if pub.ActivityTypes.Contains(it.GetType()) {
		err := pub.OnActivity(it, func(a *pub.Activity) error {
			rec := append(a.To, append(a.CC, append(a.Bto, a.BCC...)...)...)
			if !rec.Contains(pub.PublicNS) {
				if by == nil {
					by = a.Actor
				}
				if by != nil {
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
	if pub.ActorTypes.Contains(it.GetType()) {
		return id, pub.OnActor(it, func(p *pub.Actor) error {
			p.ID = id
			return nil
		})
	}
	if pub.ObjectTypes.Contains(it.GetType()) {
		return id, pub.OnObject(it, func(o *pub.Object) error {
			o.ID = id
			return nil
		})
	}
	return id, nil
}
