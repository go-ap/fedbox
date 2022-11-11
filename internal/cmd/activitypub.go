package cmd

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox"
	ap "github.com/go-ap/fedbox/activitypub"
	s "github.com/go-ap/fedbox/storage"
	"github.com/go-ap/processing"
	"github.com/urfave/cli/v2"
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
		moveObjectsCmd,
		copyObjectsCmd,
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
			name, err := loadFromStdin("Enter the actor's name")
			if err != nil {
				return errors.Errorf("Missing the actor's name")
			}
			names = append(names, string(name))
		}

		var actors = make(vocab.ItemCollection, 0)
		for _, name := range names {
			pw, err := loadPwFromStdin(true, "%s's", name)
			if err != nil {
				return err
			}
			typ := vocab.ActivityVocabularyType(c.String("type"))
			if !vocab.ActorTypes.Contains(typ) {
				typ = vocab.PersonType
			}
			self := ap.Self(vocab.IRI(ctl.Conf.BaseURL))
			now := time.Now().UTC()
			p := &vocab.Person{
				Type: typ,
				// TODO(marius): when adding authentication to the command, we can set here the actor that executes it
				AttributedTo: self.GetLink(),
				Generator:    self.GetLink(),
				Published:    now,
				Summary: vocab.NaturalLanguageValues{
					{vocab.NilLangRef, vocab.Content("Generated actor")},
				},
				Updated: now,
				PreferredUsername: vocab.NaturalLanguageValues{
					{vocab.NilLangRef, vocab.Content(name)},
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

func wrapObjectInCreate(actor vocab.Item, p vocab.Item) (vocab.Activity, error) {
	act := vocab.Activity{
		Type:    vocab.CreateType,
		To:      vocab.ItemCollection{vocab.PublicNS},
		Updated: time.Now().UTC(),
		Object:  p,
	}
	if act.AttributedTo == nil {
		act.AttributedTo = actor.GetLink()
	}
	if act.Actor == nil {
		act.Actor = actor
	}
	if !act.CC.Contains(actor.GetLink()) {
		act.CC.Append(actor.GetLink())
	}
	return act, nil
}

func (c *Control) AddObject(p *vocab.Object) (*vocab.Object, error) {
	if c.Storage == nil {
		return nil, errors.Errorf("invalid storage backend")
	}
	self, err := c.Storage.Load(vocab.IRI(c.Conf.BaseURL))
	if err != nil {
		return nil, errors.NewNotFound(err, "unable to load current's instance Service actor")
	}

	outbox := vocab.Outbox.Of(self).GetLink()
	if vocab.IsNil(outbox) {
		return nil, errors.Newf("unable to find Actor's outbox: %s", self)
	}

	create, err := wrapObjectInCreate(self, p)
	if err != nil {
		return nil, errors.Annotatef(err, "unable to wrap Object in Create activity")
	}
	if _, err := c.Saver.ProcessClientActivity(create, outbox); err != nil {
		return nil, err
	}
	return p, nil
}

func (c *Control) AddActor(p *vocab.Person, pw []byte) (*vocab.Person, error) {
	if c.Storage == nil {
		return nil, errors.Errorf("invalid storage backend")
	}
	self, err := c.Storage.Load(vocab.IRI(c.Conf.BaseURL))
	if err != nil {
		return nil, errors.NewNotFound(err, "unable to load current's instance Service actor")
	}

	create, err := wrapObjectInCreate(self, p)
	if err != nil {
		return nil, errors.Annotatef(err, "unable to wrap Actor in Create activity")
	}
	outbox := vocab.Outbox.Of(self)
	if vocab.IsNil(outbox) {
		return nil, errors.Newf("unable to find Actor's outbox: %s", self)
	}
	if _, err := c.Saver.ProcessClientActivity(create, outbox.GetLink()); err != nil {
		return nil, err
	}

	if pwManager, ok := c.Storage.(s.PasswordChanger); ok && pw != nil {
		err = pwManager.PasswordSet(p.GetLink(), pw)
	}
	return p, err
}

var ValidGenericTypes = vocab.ActivityVocabularyTypes{vocab.ObjectType, vocab.ActorType}

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
	invalidRemoveTypes := append(append(vocab.ActivityTypes, vocab.IntransitiveActivityTypes...), vocab.TombstoneType)
	self := ap.Self(vocab.IRI(c.Conf.BaseURL))

	d := new(vocab.Delete)
	d.Type = vocab.DeleteType
	d.To = vocab.ItemCollection{vocab.PublicNS}
	d.CC = make(vocab.ItemCollection, 0)
	if reason != "" {
		d.Content = vocab.NaturalLanguageValuesNew()
		d.Content.Append(vocab.NilLangRef, vocab.Content(reason))
	}
	if len(inReplyTo) > 0 {
		replIRI := make(vocab.ItemCollection, 0)
		for _, repl := range inReplyTo {
			if _, err := url.Parse(repl); err != nil {
				continue
			}
			replIRI = append(replIRI, vocab.IRI(repl))
		}
		d.InReplyTo = replIRI
	}
	d.Actor = self

	delItems := make(vocab.ItemCollection, 0)
	for _, id := range ids {
		iri := vocab.IRI(id)

		it, err := c.Storage.Load(iri)
		if err != nil {
			continue
		}
		// NOTE(marius): this should work if "it" is a collection or a single object
		vocab.OnObject(it, func(o *vocab.Object) error {
			if invalidRemoveTypes.Contains(o.GetType()) {
				return nil
			}
			d.To = o.To
			d.Bto = o.Bto
			d.CC = o.CC
			d.BCC = o.BCC
			if o.AttributedTo != nil {
				d.CC = append(d.CC, o.AttributedTo.GetLink())
			}
			delItems = append(delItems, o.GetLink())
			return nil
		})
	}
	d.CC = append(d.CC, self.GetLink())
	if len(delItems) == 0 {
		return errors.NotFoundf("No items found to delete")
	}
	d.Object = delItems

	if _, err := c.Saver.ProcessClientActivity(d, vocab.Outbox.Of(d.Actor).GetLink()); err != nil {
		return err
	}

	printItem(d, "text")
	return nil
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
	},
	Action: listObjectsAct(&ctl),
}

func printItem(it vocab.Item, outType string) error {
	if outType == "json" {
		return outJSON(it)
	}
	return outText(it)
}

func listObjectsAct(ctl *Control) cli.ActionFunc {
	return func(c *cli.Context) error {
		typeFl := c.StringSlice("type")

		var paths vocab.IRIs
		if c.NArg() == 0 {
			paths = append(paths,
				ap.ObjectsType.IRI(vocab.IRI(ctl.Conf.BaseURL)),
				ap.ActorsType.IRI(vocab.IRI(ctl.Conf.BaseURL)),
				ap.ActivitiesType.IRI(vocab.IRI(ctl.Conf.BaseURL)),
			)
		} else {
			for _, path := range c.Args().Slice() {
				u, err := url.ParseRequestURI(path)
				if err != nil {
					continue
				}
				if u.Host == "" {
					u.Host = ctl.Conf.BaseURL
				}
				paths = append(paths, vocab.IRI(u.String()))
			}
		}

		all, err := ctl.List(paths, typeFl...)
		if err != nil {
			return err
		}
		sort.Slice(all, func(i, j int) bool {
			ob1, err := vocab.ToObject(all[i])
			if err != nil {
				return false
			}
			ob2, err := vocab.ToObject(all[j])
			if err != nil {
				return true
			}
			return ob1.Published.Sub(ob2.Published) < 0
		})
		printItem(all, c.String("output"))
		return nil
	}
}

func loadPubTypes(types []string) []vocab.ActivityVocabularyType {
	objectTyp := make(vocab.ActivityVocabularyTypes, 0)
	actorTyp := make(vocab.ActivityVocabularyTypes, 0)
	activityTyp := make(vocab.ActivityVocabularyTypes, 0)
	if len(types) == 0 {
		objectTyp = vocab.ObjectTypes
		actorTyp = vocab.ActorTypes
		activityTyp = vocab.ActivityTypes
	} else {
		for _, typ := range types {
			t := vocab.ActivityVocabularyType(typ)
			if vocab.ObjectTypes.Contains(t) {
				objectTyp = append(objectTyp, t)
			}
			if vocab.ActorTypes.Contains(t) {
				actorTyp = append(actorTyp, t)
			}
			if vocab.ActivityTypes.Contains(t) {
				activityTyp = append(activityTyp, t)
			}
			if strings.ToLower(typ) == strings.ToLower(string(vocab.ObjectType)) {
				objectTyp = vocab.ObjectTypes
			}
			if strings.ToLower(typ) == strings.ToLower(string(vocab.ActorType)) {
				actorTyp = vocab.ActorTypes
			}
			if strings.ToLower(typ) == strings.ToLower(string(vocab.ActivityType)) {
				activityTyp = vocab.ActivityTypes
			}
		}
	}
	return append(append(objectTyp, actorTyp...), activityTyp...)
}

func (c *Control) List(iris vocab.IRIs, types ...string) (vocab.ItemCollection, error) {
	var typeFilter []vocab.ActivityVocabularyType
	if len(types) > 0 {
		typeFilter = loadPubTypes(types)
	}
	var items vocab.ItemCollection
	var err error
	for _, iri := range iris {
		f, _ := ap.FiltersFromIRI(iri)
		if len(typeFilter) > 0 {
			ap.Type(typeFilter...)(f)
		}
		f.MaxItems = ap.MaxItems

		col, err := c.Storage.Load(f.GetLink())

		if err != nil {
			return items, err
		}
		if col.IsCollection() {
			err = vocab.OnCollectionIntf(col, func(c vocab.CollectionInterface) error {
				items = append(items, c.Collection()...)
				return nil
			})
		} else {
			items = append(items, col)
		}
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
		typ := vocab.ActivityVocabularyType("")
		if len(f.Type) > 0 {
			typ = vocab.ActivityVocabularyType(f.Type[0].Str)
		}
		if len(f.Name) == 0 {
			if name, err := loadFromStdin("Enter the %s name", typ); err == nil {
				f.Name = append(f.Name, ap.StringEquals(string(name)))
			}
		}
		if append(vocab.ObjectTypes, vocab.ObjectType, "").Contains(typ) {
			name := f.Name[0].Str
			self := ap.Self(vocab.IRI(ctl.Conf.BaseURL))
			now := time.Now().UTC()
			p := &vocab.Object{
				Type: typ,
				// TODO(marius): when adding authentication to the command, we can set here the actor that executes it
				AttributedTo: self.GetLink(),
				Published:    now,
				Updated:      now,
				Name: vocab.NaturalLanguageValues{
					{vocab.NilLangRef, vocab.Content(name)},
				},
			}
			var err error
			if p, err = ctl.AddObject(p); err != nil {
				return errors.Annotatef(err, "Unable to save object")
			}
			fmt.Printf("Added %s [%s]: %s\n", typ, name, p.GetLink())
			return nil
		}
		return errors.Errorf("This command only supports only object types %v", vocab.ObjectTypes)
	}
}

func (c *Control) Add(types []string) (vocab.ItemCollection, error) {
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

		processor, err := processing.New(
			processing.SetIRI(vocab.IRI(baseIRI), fedbox.InternalIRI),
			processing.SetStorage(ctl.Storage),
			processing.SetLocalIRIChecker(s.IsLocalIRI(ctl.Storage)),
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
			ob, err := vocab.UnmarshalJSON(buf)
			if err != nil {
				Errf("Error unmarshaling JSON: %s", err)
				continue
			}

			col := ob
			if !ob.IsCollection() {
				col = vocab.ItemCollection{ob}
			}
			start := time.Now()
			count := 0
			vocab.OnCollectionIntf(col, func(c vocab.CollectionInterface) error {
				for _, it := range c.Collection() {
					typ := it.GetType()
					fmt.Printf("Saving %s\n", it.GetID())

					var err error
					if vocab.ActivityTypes.Contains(typ) || vocab.IntransitiveActivityTypes.Contains(typ) {
						err = vocab.OnActivity(it, func(a *vocab.Activity) error {
							if a == nil {
								return nil
							}
							_, err := processor.ProcessClientActivity(a, vocab.Outbox.Of(a.Actor).GetLink())
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

func dumpAll(f *ap.Filters) (vocab.ItemCollection, error) {
	col := make(vocab.ItemCollection, 0)
	objects, err := ctl.Storage.Load(f.GetLink())
	if err != nil {
		return col, err
	}
	if objects.IsCollection() {
		vocab.OnCollectionIntf(objects, func(c vocab.CollectionInterface) error {
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
		baseURL := vocab.IRI(ctl.Conf.BaseURL)
		objects := make(vocab.ItemCollection, 0)
		allCollections := vocab.CollectionPaths{ap.ActivitiesType, ap.ActorsType, ap.ObjectsType}
		for _, c := range allCollections {
			dump, err := dumpAll(&ap.Filters{
				IRI: vocab.IRIf(baseURL, c),
			})
			if err != nil {
				return err
			}
			objects = append(objects, dump...)
		}
		sort.Slice(objects, func(i, j int) bool {
			o1, err1 := vocab.ToObject(objects[i])
			if err1 != nil {
				return false
			}
			o2, err2 := vocab.ToObject(objects[j])
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
		objects := make(vocab.ItemCollection, 0)
		if c.Args().Len() == 0 {
			return errors.Errorf("No IRIs passed")
		}
		for i := 0; i <= c.Args().Len(); i++ {
			iri := c.Args().Get(i)
			ob, err := ctl.Storage.Load(vocab.IRI(iri))
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

var showObjectCmd = &cli.Command{Name: "show",
	Aliases:   []string{"cat"},
	Usage:     "Show an object",
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

func (c *Control) operateOnObjects(fn func(col vocab.IRI, it vocab.Item) error, to vocab.IRI, from ...vocab.Item) error {
	if !vocab.ValidCollectionIRI(to) {
		return errors.Newf("destination is not a valid collection %s", to)
	}
	_, err := c.Storage.Load(to)
	if err != nil {
		return err
	}

	for _, iri := range from {
		it, err := c.Storage.Load(iri.GetLink())
		if err != nil {
			return err
		}
		if vocab.IsItemCollection(it) {
			return vocab.OnCollectionIntf(it, func(col vocab.CollectionInterface) error {
				return c.operateOnObjects(fn, to, col.Collection()...)
			})
		}
		if !vocab.IsObject(it) {
			return errors.Newf("Invalid object at IRI %s, %v", from, it)
		}

		if err = fn(to, it); err != nil {
			return err
		}
	}

	return nil
}

var moveObjectsCmd = &cli.Command{
	Name:    "move",
	Aliases: []string{"mv"},
	Usage:   "Move ActivityPub objects\nUsage: activitypub move IRI CollectionIRI",
	Action:  movePubObjects(&ctl),
}

func movePubObjects(ctl *Control) cli.ActionFunc {
	return func(c *cli.Context) error {
		if c.NArg() < 2 {
			return errors.Errorf("Need a source IRI and a destination collection IRI")
		}
		source := vocab.IRI(c.Args().Get(0))
		destination := vocab.IRI(c.Args().Get(1))
		return ctl.MoveObjects(source, destination)
	}
}

func (c *Control) MoveObjects(to vocab.IRI, from ...vocab.Item) error {
	st, ok := c.Storage.(processing.CollectionStore)
	if !ok {
		return errors.Newf("invalid storage %T", c.Storage)
	}

	copyFn := func(col vocab.IRI, it vocab.Item) error {
		if err := st.AddTo(col.GetLink(), it); err != nil {
			return err
		}

		if err := c.Storage.Delete(it.GetLink()); err != nil {
			return err
		}
		return nil
	}
	return c.operateOnObjects(copyFn, to, from...)
}

var copyObjectsCmd = &cli.Command{
	Name:    "copy",
	Aliases: []string{"cp"},
	Usage:   "Copy ActivityPub objects\nUsage: activitypub copy IRI CollectionIRI",
	Action:  copyPubObjects(&ctl),
}

func copyPubObjects(ctl *Control) cli.ActionFunc {
	return func(c *cli.Context) error {
		if c.NArg() < 2 {
			return errors.Errorf("Need a source IRI and a destination collection IRI")
		}
		argSl := c.Args().Slice()
		var source vocab.ItemCollection
		for _, arg := range argSl[:c.NArg()-1] {
			source = append(source, vocab.IRI(arg))
		}
		destination := vocab.IRI(argSl[c.NArg()-1])
		return ctl.CopyObjects(destination, source...)
	}
}

func (c *Control) CopyObjects(to vocab.IRI, from ...vocab.Item) error {
	st, ok := c.Storage.(processing.CollectionStore)
	if !ok {
		return errors.Newf("invalid storage %T", c.Storage)
	}

	copyFn := func(col vocab.IRI, it vocab.Item) error {
		err := st.AddTo(col.GetLink(), it)
		if err != nil {
			Errf("Error: %s", err)
		}
		return nil
	}
	return c.operateOnObjects(copyFn, to, from...)
}
