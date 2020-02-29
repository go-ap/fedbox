package cmd

import (
	"fmt"
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/handlers"
	"github.com/go-ap/storage"
	"gopkg.in/urfave/cli.v2"
	"net/url"
	"time"
)

var Actors = &cli.Command{
	Name:  "actor",
	Usage: "Actor management helper",
	Subcommands: []*cli.Command{
		addActor,
		delActor,
		listActors,
	},
}

var addActor = &cli.Command{
	Name:    "add",
	Aliases: []string{"new"},
	Usage:   "Adds an ActivityPub actor",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "type",
			Usage: fmt.Sprintf("The type of activitypub actor to add"),
		},
	},
	Action: AddActor(&ctl),
}

var delActor = &cli.Command{
	Name:    "del",
	Aliases: []string{"delete", "remove", "rm"},
	Usage:   "Deletes an ActivityPub actor",
	Action:  DelActor(&ctl),
}

var listActors = &cli.Command{
	Name:    "ls",
	Aliases: []string{"list"},
	Usage:   "Lists existing actors",
	Action:  ListActors(&ctl),
}

func AddActor(ctl *Control) cli.ActionFunc {
	return func(c *cli.Context) error {
		names := c.Args().Slice()

		var actors = make(pub.ItemCollection, 0)
		for _, name := range names {

			pw, err := loadPwFromStdin(true, "%s's", name)
			if err != nil {
				return err
			}
			typ := pub.ActivityVocabularyType(c.String("type"))
			if !pub.ActorTypes.Contains(typ) {
				typ = pub.PersonType
			}
			p, err := ctl.AddActor(name, typ, nil, pw)
			if err != nil {
				Errf("Error adding %s: %s\n", name, err)
			}
			fmt.Printf("Added %q [%s]: %s\n", typ, name, p.GetLink())
			actors = append(actors, p)
		}
		return nil
	}
}

func DelActor(ctl *Control) cli.ActionFunc {
	return func(c *cli.Context) error {
		ids := c.Args().Slice()

		for _, id := range ids {
			err := ctl.DeleteActor(id)
			if err != nil {
				Errf("Error deleting %s: %s\n", id, err)
				continue
			}
			fmt.Printf("Deleted: %s\n", id)
		}
		return nil
	}
}

func ListActors(ctl *Control) cli.ActionFunc {
	return func(c *cli.Context) error {
		actors, err := ctl.ListActors()
		if err != nil {
			return err
		}
		for i, it := range actors {
			if act, err := pub.ToActor(it); err != nil {
				fmt.Printf("%3d [%11s] %s\n", i, it.GetType(), it.GetLink())
			} else {
				fmt.Printf("%3d [%11s] %s\n%s\n", i, it.GetType(), act.PreferredUsername.First(), it.GetLink())
			}
		}
		return nil
	}
}

type PasswordChanger interface {
	PasswordSet(pub.Item, []byte) error
	PasswordCheck(pub.Item, []byte) error
}

func (c *Control) AddActor(preferredUsername string, typ pub.ActivityVocabularyType, id *pub.ID, pw []byte) (*pub.Person, error) {
	self := ap.Self(pub.IRI(c.BaseURL))
	now := time.Now().UTC()
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
		return errors.Newf("nothing found")
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

var ValidGenericTypes = pub.ActivityVocabularyTypes{pub.ObjectType, pub.ActorType}

func (c *Control) Delete(id, typ string) error {
	t := pub.ActivityVocabularyType(typ)
	if !(pub.ActorTypes.Contains(t) || pub.ObjectTypes.Contains(t) || ValidGenericTypes.Contains(t)) {
		return errors.Errorf("invalid ActivityPub object type %s", typ)
	}

	var iri pub.IRI
	var loadFn func(storage.Filterable) (pub.ItemCollection, uint, error)
	var delFn func(pub.Item) (pub.Item, error)

	var col string
	if pub.ActorTypes.Contains(t) || t == pub.ActorType {
		col = "actors"
		loadFn = c.Storage.LoadActors
		delFn = c.Storage.DeleteActor
	}
	if pub.ObjectTypes.Contains(t) || t == pub.ObjectType {
		col = "objects"
		loadFn = c.Storage.LoadObjects
		delFn = c.Storage.DeleteObject
	}

	if u, err := url.Parse(id); err != nil {
		self := ap.Self(pub.IRI(c.BaseURL))
		iri = pub.IRI(fmt.Sprintf("%s/%s/%s", self.ID, col, id))
	} else {
		iri = pub.IRI(u.String())
	}
	it, cnt, err := loadFn(iri)
	if err != nil {
		return err
	}
	if cnt == 0 {
		return errors.Newf("nothing found")
	}
	_, err = delFn(it.First())

	return err
}

func (c *Control) List(f storage.Filterable) error {
	return nil
}
