package cmd

import (
	"fmt"
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/handlers"
	"github.com/go-ap/storage"
	"github.com/openshift/osin"
	"net/url"
	"time"
)

type Control struct {
	Conf        config.Options
	Host        string
	BaseURL     string
	AuthStorage osin.Storage
	Storage     storage.Repository
}

type PasswordChanger interface {
	PasswordSet(pub.Item, []byte) error
	PasswordCheck(pub.Item, []byte) error
}

func (c *Control) AddActor(preferredUsername string, typ pub.ActivityVocabularyType, id *pub.ID, pw []byte) (*pub.Person, error) {
	self := ap.Self(pub.IRI(c.BaseURL))
	now := time.Now()
	p := pub.Person{
		Type: typ,
		// TODO(marius): when adding authentication to the command, we can set here the actor that executes it
		AttributedTo: self.GetLink(),
		Audience:     pub.ItemCollection{pub.PublicNS},
		Generator:    self.GetLink(),
		Published:    now,
		Summary: pub.NaturalLanguageValues{
			{pub.NilLangRef, "Generated actor"},
		},
		Updated: now,
		PreferredUsername: pub.NaturalLanguageValues{
			{pub.NilLangRef, preferredUsername},
		},
	}

	// TODO(marius): add annotations for the errors
	if id == nil {
		if gen, ok := c.Storage.(storage.IDGenerator); ok {
			newId, err := gen.GenerateID(p, self)
			if err != nil {
				return nil, err
			}
			id = &newId
		}
	}
	p.ID = *id
	p.URL = p.GetLink()
	p.Inbox = pub.IRI(fmt.Sprintf("%s/%s", p.ID, handlers.Inbox))
	p.Outbox = pub.IRI(fmt.Sprintf("%s/%s", p.ID, handlers.Outbox))
	p.Liked = pub.IRI(fmt.Sprintf("%s/%s", p.ID, handlers.Liked))
	p.Followers = pub.IRI(fmt.Sprintf("%s/%s", p.ID, handlers.Followers))
	p.Following = pub.IRI(fmt.Sprintf("%s/%s", p.ID, handlers.Following))

	p.Endpoints = &pub.Endpoints{
		SharedInbox:                self.Inbox.GetLink(),
		OauthAuthorizationEndpoint: pub.IRI(fmt.Sprintf("%s/oauth/authorize", self.URL)),
		OauthTokenEndpoint:         pub.IRI(fmt.Sprintf("%s/oauth/token", self.URL)),
	}
	it, err := c.Storage.SaveActor(p)
	if err != nil {
		return nil, err
	}

	saved, err := pub.ToActor(it)
	if err != nil {
		return nil, err
	}

	if pw != nil {
		if pwManager, ok := c.Storage.(PasswordChanger); ok {
			err := pwManager.PasswordSet(saved.GetLink(), pw)
			if err != nil {
				return saved, err
			}
		}
	}

	return saved, nil
}

func (c *Control) DeleteActor(id string) error {
	self := ap.Self(pub.IRI(c.BaseURL))
	var iri pub.IRI
	if u, err := url.Parse(id); err != nil {
		iri = pub.IRI(fmt.Sprintf("%s/%s/%s", self.ID, ap.ActorsType, id))
	} else {
		iri = pub.IRI(u.String())
	}
	it, cnt, err := c.Storage.LoadActors(iri)
	if err != nil {
		return err
	}
	if cnt == 0 {
		return errors.Newf("")
	}
	_, err = c.Storage.DeleteActor(it.First())
	return err
}

func (c *Control) ListActors() (pub.ItemCollection, error) {
	var err error
	actorsIRI := pub.IRI(fmt.Sprintf("%s/%s", c.BaseURL, ap.ActorsType))
	col, _, err := c.Storage.LoadActors(&ap.Filters{IRI: actorsIRI})
	if err != nil {
		return col, errors.Annotatef(err, "Unable to load actors")
	}
	return col, nil
}
