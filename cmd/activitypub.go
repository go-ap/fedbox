package cmd

import (
	"fmt"
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	s "github.com/go-ap/fedbox/storage"
	"github.com/go-ap/handlers"
	"github.com/go-ap/storage"
	"gopkg.in/urfave/cli.v2"
	"net/url"
	"path"
	"strings"
	"time"
)

var Pub = &cli.Command{
	Name:    "pub",
	Aliases: []string{"ap"},
	Usage:   "ActivityPub management helper",
	Subcommands: []*cli.Command{
		actors,
		listObjects,
		delObjects,
	},
}

var actors = &cli.Command{
	Name:  "actor",
	Usage: "Actor management helper",
	Subcommands: []*cli.Command{
		addActor,
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
	Action: addActorAct(&ctl),
}

func addActorAct(ctl *Control) cli.ActionFunc {
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
		if pwManager, ok := c.Storage.(s.PasswordChanger); ok {
			err := pwManager.PasswordSet(saved.GetLink(), pw)
			if err != nil {
				return saved, err
			}
		}
	}

	return saved, nil
}

var ValidGenericTypes = pub.ActivityVocabularyTypes{pub.ObjectType, pub.ActorType}

var delObjects = &cli.Command{
	Name:    "delete",
	Aliases: []string{"del", "rm"},
	Usage:   "Deletes an ActivityPub object",
	Flags: []cli.Flag{
		&cli.StringSliceFlag{
			Name:        "type",
			Usage:       fmt.Sprintf("The type of ActivityPub object"),
			DefaultText: fmt.Sprintf("Valid values: %v", ValidGenericTypes),
		},
	},
	Action: delObjectsAct(&ctl),
}

func delObjectsAct(ctl *Control) cli.ActionFunc {
	return func(c *cli.Context) error {
		ids := c.Args().Slice()

		var err error
		for _, id := range ids {
			err := ctl.DeleteObject(id)
			if err != nil {
				Errf("Error deleting %s: %s\n", id, err)
				continue
			}
			fmt.Printf("Deleted: %s\n", id)
		}
		if err != nil {
			return err
		}
		return nil
	}
}

func (c *Control) DeleteObject(id string) error {
	u, err := url.Parse(id)
	if err != nil {
		return err
	}
	base, _ := path.Split(u.Path)
	typ := strings.Trim(base, "/")

	var loadFn func(storage.Filterable) (pub.ItemCollection, uint, error)
	var delFn func(pub.Item) (pub.Item, error)

	if strings.ToLower(typ) == strings.ToLower(string(ap.ActorsType)) {
		loadFn = c.Storage.LoadActors
		delFn = c.Storage.DeleteActor
	} else if strings.ToLower(typ) == strings.ToLower(string(ap.ObjectsType)) {
		loadFn = c.Storage.LoadObjects
		delFn = c.Storage.DeleteObject
	} else {
		return errors.Errorf("invalid ActivityPub object type %s", typ)
	}

	it, cnt, err := loadFn(pub.IRI(id))
	if err != nil {
		return err
	}
	if cnt == 0 {
		return errors.Newf("nothing found")
	}
	_, err = delFn(it.First())

	return err
}

var listObjects = &cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Usage:   "Lists objects",
	Flags: []cli.Flag{
		&cli.StringSliceFlag{
			Name:        "type",
			Usage:       fmt.Sprintf("The type of activitypub object to list"),
			DefaultText: fmt.Sprintf("Valid values: %v", ValidGenericTypes),
		},
	},
	Action: listObjectsAct(&ctl),
}

func printItem(it pub.Item) {
	typ := it.GetType()
	if pub.ObjectTypes.Contains(typ) {
	}
	if pub.ActorTypes.Contains(typ) {
	}
	if pub.ActivityTypes.Contains(typ) {
	}
}

func listObjectsAct(ctl *Control) cli.ActionFunc {
	return func(c *cli.Context) error {
		typeFl := c.StringSlice("type")
		all, err := ctl.List(typeFl)
		if err != nil {
			return err
		}
		for i, it := range all {
			fmt.Printf("%4d [%11s] %s\n", i, it.GetType(), it.GetLink())
		}
		return nil
	}
}

func loadPubTypes(types []string) (pub.ActivityVocabularyTypes, pub.ActivityVocabularyTypes, pub.ActivityVocabularyTypes) {
	objectTyp := make(pub.ActivityVocabularyTypes, 0)
	actorTyp := make(pub.ActivityVocabularyTypes, 0)
	activityTyp := make(pub.ActivityVocabularyTypes, 0)
	if len(types) == 0 {
		objectTyp = pub.ObjectTypes
		actorTyp = pub.ActorTypes
		activityTyp = pub.ActivityTypes
	} else {
		for _, typ := range types {
			t := pub.ActivityVocabularyType(typ)
			if pub.ObjectTypes.Contains(t) {
				objectTyp = append(objectTyp, t)
			}
			if pub.ActorTypes.Contains(t) {
				actorTyp = append(actorTyp, t)
			}
			if pub.ActivityTypes.Contains(t) {
				activityTyp = append(activityTyp, t)
			}
			if strings.ToLower(typ) == strings.ToLower(string(pub.ObjectType)) {
				objectTyp = pub.ObjectTypes
			}
			if strings.ToLower(typ) == strings.ToLower(string(pub.ActorType)) {
				actorTyp = pub.ActorTypes
			}
			if strings.ToLower(typ) == strings.ToLower(string(pub.ActivityType)) {
				activityTyp = pub.ActivityTypes
			}
		}
	}
	return objectTyp, actorTyp, activityTyp
}

func (c *Control) List(types []string) (pub.ItemCollection, error) {
	var items pub.ItemCollection
	objectTyp, actorTyp, activityTyp := loadPubTypes(types)
	accFn := func(colTyp handlers.CollectionType, types pub.ActivityVocabularyTypes) error {
		if len(types) == 0 {
			return nil
		}
		baseIRI := pub.IRI(fmt.Sprintf("%s/%s", ctl.BaseURL, colTyp))
		f := ap.FiltersNew(
			ap.IRI(baseIRI),
			ap.Type(types...),
		)
		col, err := c.Storage.LoadCollection(f)
		if err != nil {
			return err
		}

		err = pub.OnCollectionIntf(col, func(c pub.CollectionInterface) error {
			for _, tt := range c.Collection() {
				items = append(items, tt)
			}
			return nil
		})
		return nil
	}
	err := accFn(ap.ObjectsType, objectTyp)
	if err != nil {
		return items, err
	}
	err = accFn(ap.ActorsType, actorTyp)
	if err != nil {
		return items, err
	}
	err = accFn(ap.ActivitiesType, activityTyp)
	return items, err
}
