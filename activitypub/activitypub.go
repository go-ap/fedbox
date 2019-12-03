package activitypub

import (
	"fmt"
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	"github.com/go-ap/handlers"
	"github.com/pborman/uuid"
	"net/url"
	"path"
	"strings"
)

func Self(baseURL pub.IRI) pub.Service {
	url, _ := baseURL.URL()
	inbox := *url
	inbox.Path = path.Join(inbox.Path, string(handlers.Inbox))

	oauth := *url
	oauth.Path = path.Join(oauth.Path, "oauth/")
	return pub.Service{
		ID:           pub.ObjectID(url.String()),
		Type:         pub.ServiceType,
		Name:         pub.NaturalLanguageValues{{Ref: pub.NilLangRef, Value: "self"}},
		AttributedTo: pub.IRI("https://github.com/mariusor"),
		Audience:     pub.ItemCollection{pub.PublicNS},
		Content:      nil, //pub.NaturalLanguageValues{{Ref: pub.NilLangRef, Value: ""}},
		Icon:         nil,
		Image:        nil,
		Location:     nil,
		Summary:      pub.NaturalLanguageValues{{Ref: pub.NilLangRef, Value: "Generic ActivityPub service"}},
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

// ItemByType
func ItemByType(typ pub.ActivityVocabularyType) (pub.Item, error) {
	if pub.ActorTypes.Contains(typ) {
		return &pub.Actor{Type: typ}, nil
	} else if pub.ActivityTypes.Contains(typ) {
		return &pub.Activity{Type: typ}, nil
	} else if typ == pub.CollectionType {
		return &Collection{Type: typ}, nil
	} else if typ == pub.OrderedCollectionType {
		return &OrderedCollection{Type: typ}, nil
	}
	return pub.JSONGetItemByType(typ)
}

// ToOrderedCollection
func ToOrderedCollection(it pub.Item) (*OrderedCollection, error) {
	switch o := it.(type) {
	case *OrderedCollection:
		return o, nil
	case OrderedCollection:
		return &o, nil
	case *pub.OrderedCollection:
		col := OrderedCollection(*o)
		return &col, nil
	case pub.OrderedCollection:
		col := OrderedCollection(o)
		return &col, nil
	}
	return nil, errors.Newf("invalid ordered collection")
}

// ToCollection
func ToCollection(it pub.Item) (*Collection, error) {
	switch o := it.(type) {
	case *Collection:
		return o, nil
	case Collection:
		return &o, nil
	case *pub.Collection:
		col := Collection(*o)
		return &col, nil
	case pub.Collection:
		col := Collection(o)
		return &col, nil
	}
	return nil, errors.Newf("invalid  collection")
}

// GenerateID generates an unique identifier for the it ActivityPub Object.
func GenerateID(it pub.Item, partOf string, by pub.Item) (pub.ObjectID, error) {
	uuid := uuid.New()
	id := pub.ObjectID(fmt.Sprintf("%s/%s", strings.ToLower(partOf), uuid))
	if pub.ActivityTypes.Contains(it.GetType()) {
		return id, pub.OnActivity(it, func(a *pub.Activity) error {
			a.ID = id
			return nil
		})
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
