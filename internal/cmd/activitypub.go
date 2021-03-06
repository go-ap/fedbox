package cmd

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/app"
	s "github.com/go-ap/fedbox/storage"
	"github.com/go-ap/handlers"
	"github.com/go-ap/processing"
	"gopkg.in/urfave/cli.v2"
)

var PubCmd = &cli.Command{
	Name:    "pub",
	Aliases: []string{"ap"},
	Usage:   "ActivityPub management helper",
	Subcommands: []*cli.Command{
		actorsCmd,
		addObjectCmd,
		listObjectsCmd,
		showObjectCmd,
		delObjectsCmd,
		exportCmd,
		importCmd,
	},
}

var actorsCmd = &cli.Command{
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
		if len(names) == 0 {
			return errors.Errorf("Missing actor name")
		}

		var actors = make(pub.ItemCollection, 0)
		for _, name := range names {
			pw, err := loadPwFromStdin(true, "%s's", name)
			if err != nil {
				Errf(err.Error())
				return err
			}
			typ := pub.ActivityVocabularyType(c.String("type"))
			if !pub.ActorTypes.Contains(typ) {
				typ = pub.PersonType
			}
			self := ap.Self(pub.IRI(ctl.Conf.BaseURL))
			now := time.Now().UTC()
			p := &pub.Person{
				Type: typ,
				// TODO(marius): when adding authentication to the command, we can set here the actor that executes it
				AttributedTo: self.GetLink(),
				Generator:    self.GetLink(),
				Published:    now,
				Summary: pub.NaturalLanguageValues{
					{pub.NilLangRef, pub.Content("Generated actor")},
				},
				Updated: now,
				PreferredUsername: pub.NaturalLanguageValues{
					{pub.NilLangRef, pub.Content(name)},
				},
			}
			if p, err = ctl.AddActor(p, pw); err != nil {
				Errf("Error adding %s: %s\n", name, err)
			}
			fmt.Printf("Added %q [%s]: %s\n", typ, name, p.GetLink())
			if metaSaver, ok := ctl.Storage.(s.MetadataTyper); ok {
				if err := AddKeyToItem(metaSaver, p); err != nil {
					Errf("Error saving metadata for %s: %s", name, err)
				}
			}
			actors = append(actors, p)
		}
		return nil
	}
}

func (c *Control) AddActor(p *pub.Person, pw []byte) (*pub.Person, error) {
	if c.Storage == nil {
		return nil, errors.Errorf("invalid storage backend")
	}
	self := ap.Self(pub.IRI(c.Conf.BaseURL))
	now := time.Now().UTC()

	act := &pub.Activity{
		Type:         pub.CreateType,
		AttributedTo: self.GetLink(),
		Actor:        self.GetLink(),
		To:           pub.ItemCollection{pub.PublicNS},
		CC:           pub.ItemCollection{self.ID},
		Updated:      now,
		Object:       p,
	}

	var err error
	_, err = c.Saver.ProcessClientActivity(act)
	if err != nil {
		return nil, err
	}

	if pw == nil {
		return p, nil
	}
	if pwManager, ok := c.Storage.(s.PasswordChanger); ok {
		if err := pwManager.PasswordSet(p.GetLink(), pw); err != nil {
			return p, err
		}
	}

	return p, nil
}

var ValidGenericTypes = pub.ActivityVocabularyTypes{pub.ObjectType, pub.ActorType}

var delObjectsCmd = &cli.Command{
	Name:    "delete",
	Aliases: []string{"del", "rm"},
	Usage:   "Deletes an ActivityPub object",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "reason",
			Usage: fmt.Sprintf("The reason why we want to delete the item"),
		},
		&cli.StringSliceFlag{
			Name:  "inReplyTo",
			Usage: fmt.Sprintf("If deletion is a followup on moderation activities"),
		},
	},
	Action: delObjectsAct(&ctl),
}

func delObjectsAct(ctl *Control) cli.ActionFunc {
	return func(c *cli.Context) error {
		return ctl.DeleteObjects(c.String("reason"), c.StringSlice("inReplyTo"), c.Args().Slice()...)
	}
}

func (c *Control) DeleteObjects(reason string, inReplyTo []string, ids ...string) error {
	self := ap.Self(pub.IRI(c.Conf.BaseURL))

	d := new(pub.Delete)
	d.Type = pub.DeleteType
	d.To = pub.ItemCollection{pub.PublicNS}
	d.CC = make(pub.ItemCollection, 0)
	if reason != "" {
		d.Content = pub.NaturalLanguageValuesNew()
		d.Content.Append(pub.NilLangRef, pub.Content(reason))
	}
	if len(inReplyTo) > 0 {
		replIRI := make(pub.ItemCollection, 0)
		for _, repl := range inReplyTo {
			if _, err := url.Parse(repl); err != nil {
				continue
			}
			replIRI = append(replIRI, pub.IRI(repl))
		}
		d.InReplyTo = replIRI
	}
	d.Actor = self
	d.CC = append(d.CC, self.GetLink())

	delItems := make(pub.ItemCollection, 0)
	for _, id := range ids {
		iri := pub.IRI(id)
		u, err := iri.URL()
		if err != nil {
			continue
		}
		base, _ := path.Split(u.Path)
		typ := strings.Trim(base, "/")

		var it pub.Item
		if strings.ToLower(typ) != strings.ToLower(string(ap.ActorsType)) && strings.ToLower(typ) != strings.ToLower(string(ap.ObjectsType)) {
			continue
		}
		it, err = c.Storage.Load(iri)
		if err != nil || it.GetType() == pub.TombstoneType {
			continue
		}
		if it.IsCollection() {
			pub.OnCollectionIntf(it, func(c pub.CollectionInterface) error {
				for _, ob := range c.Collection() {
					pub.OnObject(ob, func(o *pub.Object) error {
						if o.AttributedTo != nil {
							d.CC = append(d.CC, o.AttributedTo.GetLink())
						}
						return nil
					})
					delItems = append(delItems, ob.GetLink())
				}
				return nil
			})
		}
	}
	if len(delItems) == 0 {
		return errors.NotFoundf("No items found to delete")
	}
	d.Object = delItems

	act, err := c.Saver.ProcessClientActivity(d)

	printItem(act, "text")
	return err
}

var listObjectsCmd = &cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Usage:   "Lists objects",
	Flags: []cli.Flag{
		&cli.StringSliceFlag{
			Name:        "type",
			Usage:       fmt.Sprintf("The type of activitypub object to list"),
			DefaultText: fmt.Sprintf("Valid values: %v", ValidGenericTypes),
		},
		&cli.StringFlag{
			Name:        "output",
			Usage:       fmt.Sprintf("The format in which to output the items."),
			DefaultText: fmt.Sprintf("Valid values: %v", []string{"json", "text"}),
			Value:       "text",
		},
		&cli.StringFlag{
			Name:  "path",
			Usage: "Pass the URL path at which to start from.",
			Value: "/",
		},
	},
	Action: listObjectsAct(&ctl),
}

func printItem(it pub.Item, outType string) error {
	if outType == "json" {
		return outJSON(it)
	}
	return outText(it)
}

func listObjectsAct(ctl *Control) cli.ActionFunc {
	return func(c *cli.Context) error {
		initialPath := c.String("path")
		typeFl := c.StringSlice("type")
		all, err := ctl.List(initialPath, typeFl...)
		if err != nil {
			return err
		}
		sort.Slice(all, func(i, j int) bool {
			ob1, err := pub.ToObject(all[i])
			if err != nil {
				return false
			}
			ob2, err := pub.ToObject(all[j])
			if err != nil {
				return true
			}
			return ob1.Published.Sub(ob2.Published) < 0
		})
		printItem(all, c.String("output"))
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

func (c *Control) List(initialPath string, types ...string) (pub.ItemCollection, error) {
	var items pub.ItemCollection
	objectTyp, actorTyp, activityTyp := loadPubTypes(types)
	accFn := func(baseIRI pub.IRI, types pub.ActivityVocabularyTypes) error {
		if len(types) == 0 {
			return nil
		}
		f := ap.FiltersNew(
			ap.BaseIRI(handlers.Split(baseIRI)),
			ap.Type(types...),
		)
		col, err := c.Storage.Load(f.GetLink())
		if err != nil {
			return err
		}

		if col.IsCollection() {
			err = pub.OnCollectionIntf(col, func(c pub.CollectionInterface) error {
				items = append(items, c.Collection()...)
				return nil
			})
	 	} else {
	 		items = append(items, col)
		}
		return nil
	}
	var err error
	if initialPath == "" || initialPath == "/" {
		obIRI := ap.ObjectsType.IRI(pub.IRI(ctl.Conf.BaseURL))
		err = accFn(obIRI, objectTyp)
		if err != nil {
			return items, err
		}
		actorsIRI := ap.ActorsType.IRI(pub.IRI(ctl.Conf.BaseURL))
		err = accFn(actorsIRI, actorTyp)
		if err != nil {
			return items, err
		}
		activitiesIRI := ap.ActivitiesType.IRI(pub.IRI(ctl.Conf.BaseURL))
		err = accFn(activitiesIRI, activityTyp)
	} else {
		if initialPath[0] != '/' {
			initialPath = "/" + initialPath
		}
		err = accFn(pub.IRI(ctl.Conf.BaseURL).AddPath(initialPath), activityTyp)
	}
	return items, err
}

var addObjectCmd = &cli.Command{
	Name:    "add",
	Aliases: []string{"new"},
	Usage:   "Adds a new object",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "type",
			Usage:       fmt.Sprintf("The type of activitypub object to create"),
			DefaultText: fmt.Sprintf("Valid values: %v", ValidGenericTypes),
		},
	},
	Action: addObjectAct(&ctl),
}

func addObjectAct(ctl *Control) cli.ActionFunc {
	return func(c *cli.Context) error {
		f, _ := LoadFilters(c)
		typ := f.Type[0]
		if pub.ActorTypes.Contains(pub.ActivityVocabularyType(typ.Str)) {
			name, err := loadFromStdin("Enter the %s name", typ)
			pw, err := loadPwFromStdin(true, "%s's", name)
			if err != nil {
				return err
			}

			self := ap.Self(pub.IRI(ctl.Conf.BaseURL))
			now := time.Now().UTC()
			p := &pub.Person{
				Type: pub.ActivityVocabularyType(typ.Str),
				// TODO(marius): when adding authentication to the command, we can set here the actor that executes it
				AttributedTo: self.GetLink(),
				Generator:    self.GetLink(),
				Published:    now,
				Summary: pub.NaturalLanguageValues{
					{pub.NilLangRef, pub.Content("Generated actor")},
				},
				Updated: now,
				PreferredUsername: pub.NaturalLanguageValues{
					{pub.NilLangRef, pub.Content(name)},
				},
			}
			if p, err = ctl.AddActor(p, pw); err != nil {
				Errf("Error adding %s: %s\n", name, err)
			}
			fmt.Printf("Added %s [%s]: %s\n", typ, name, p.GetLink())
			return nil
		}
		return errors.Errorf("Unknown type %s", typ)
	}
}

func (c *Control) Add(types []string) (pub.ItemCollection, error) {
	return nil, nil
}

var importCmd = &cli.Command{
	Name:    "import",
	Aliases: []string{"load"},
	Usage:   "Imports ActivityPub objects",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "base",
			Usage: fmt.Sprintf("The base IRI to replace"),
		},
	},
	Action: importPubObjects(&ctl),
}

func importPubObjects(ctl *Control) cli.ActionFunc {
	return func(c *cli.Context) error {
		baseIRI := ctl.Conf.BaseURL
		toReplace := c.String("base")
		files := c.Args().Slice()

		processor, _, err := processing.New(
			processing.SetIRI(pub.IRI(baseIRI), app.InternalIRI),
			processing.SetStorage(ctl.Storage),
		)
		if err != nil {
			Errf("Error initializing ActivityPub processor: %s", err)
			return err
		}
		for _, name := range files {
			f, err := os.Open(name)
			if err != nil {
				if os.IsNotExist(err) {
					Errf("Invalid path %s", name)
				} else {
					Errf("Error %s", err)
				}
			}

			s, err := f.Stat()
			if err != nil {
				Errf("Error %s", err)
				continue
			}
			buf := make([]byte, s.Size())
			size, err := f.Read(buf)
			if err != nil {
				Errf("Error %s", err)
				continue
			}
			if size == 0 {
				Errf("Empty file %s", name)
				continue
			}

			if len(toReplace) > 0 {
				buf = bytes.Replace(buf, []byte(toReplace), []byte(baseIRI), -1)
			}
			ob, err := pub.UnmarshalJSON(buf)
			if err != nil {
				Errf("Error unmarshaling JSON: %s", err)
				continue
			}

			col := ob
			if !ob.IsCollection() {
				col = pub.ItemCollection{ob}
			}
			start := time.Now()
			count := 0
			pub.OnCollectionIntf(col, func(c pub.CollectionInterface) error {
				for _, it := range c.Collection() {
					typ := it.GetType()
					fmt.Printf("Saving %s\n", it.GetID())

					var err error
					if pub.ActivityTypes.Contains(typ) || pub.IntransitiveActivityTypes.Contains(typ) {
						err = pub.OnActivity(it, func(a *pub.Activity) error {
							if a == nil {
								return nil
							}
							_, err := processor.ProcessClientActivity(a)
							return err
						})
					} else {
						it, err = ctl.Storage.Save(it)
					}
					if err != nil {
						Errf("Unable to save %s %s: %s", it.GetType(), it.GetID(), err)
						continue
					}
				}
				return nil
			})
			tot := time.Now().Sub(start)
			fmt.Printf("Ellapsed time:          %s\n", tot)
			if count > 0 {
				perIt := time.Duration(int64(tot) / int64(count))
				fmt.Printf("Ellapsed time per item: %s\n", perIt)
			}
		}
		return nil
	}
}

var exportCmd = &cli.Command{
	Name:    "export",
	Aliases: []string{"dump"},
	Usage:   "Exports ActivityPub objects",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "output",
			Usage:       fmt.Sprintf("The format in which to output the items (%v)", []string{"json", "text"}),
			DefaultText: "Default: json",
			Value:       "json",
		},
	},
	Action: exportPubObjects(&ctl),
}

func dumpAll(f *ap.Filters) (pub.ItemCollection, error) {
	col := make(pub.ItemCollection, 0)
	objects, err := ctl.Storage.Load(f.GetLink())
	if err != nil {
		return col, err
	}
	if objects.IsCollection() {
		pub.OnCollectionIntf(objects, func(c pub.CollectionInterface) error {
			col = append(col, c.Collection()...)
			return nil
		})
	} else {
		col = append(col, objects)
	}
	return col, nil
}

func exportPubObjects(ctl *Control) cli.ActionFunc {
	return func(c *cli.Context) error {
		baseURL := pub.IRI(ctl.Conf.BaseURL)
		objects := make(pub.ItemCollection, 0)
		allCollections := handlers.CollectionTypes{ap.ActivitiesType, ap.ActorsType, ap.ObjectsType}
		for _, c := range allCollections {
			dump, err := dumpAll(&ap.Filters{
				IRI: handlers.IRIf(baseURL, c),
			})
			if err != nil {
				return err
			}
			objects = append(objects, dump...)
		}
		sort.Slice(objects, func(i, j int) bool {
			o1, err1 := pub.ToObject(objects[i])
			if err1 != nil {
				return false
			}
			o2, err2 := pub.ToObject(objects[j])
			if err2 != nil {
				return false
			}
			return o1.Published.Sub(o2.Published) < 0
		})
		return printItem(objects, c.String("output"))
	}
}

func showObjectAct(ctl *Control) cli.ActionFunc {
	return func(c *cli.Context) error {
		objects := make(pub.ItemCollection, 0)
		if c.Args().Len() == 0 {
			return errors.Errorf("No IRIs passed")
		}
		for i := 0; i <= c.Args().Len(); i++ {
			iri := c.Args().Get(i)
			ob, err := ctl.Storage.Load(pub.IRI(iri))
			if err != nil {
				Errf(err.Error())
				continue
			}
			objects = append(objects, ob)
		}

		for _, ob := range objects {
			printItem(ob, c.String("output"))
		}
		return nil
	}
}

var showObjectCmd = &cli.Command{
	Name:    "show",
	Aliases: []string{"cat"},
	Usage:   "Show an object",
	ArgsUsage: "IRI...",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "output",
			Usage:       fmt.Sprintf("The format in which to output the items."),
			DefaultText: fmt.Sprintf("Valid values: %v", []string{"json", "text"}),
			Value:       "text",
		},
	},
	Action: showObjectAct(&ctl),
}
