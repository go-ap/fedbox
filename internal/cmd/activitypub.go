package cmd

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"syscall"
	"time"

	"git.sr.ht/~mariusor/lw"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/auth"
	c "github.com/go-ap/client"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox"
	ap "github.com/go-ap/fedbox/activitypub"
	s "github.com/go-ap/fedbox/storage"
	"github.com/go-ap/filters"
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
		indexCmd,
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
		&cli.StringFlag{
			Name:  "key-type",
			Usage: fmt.Sprintf("Type of keys to generate: %v", []string{fedbox.KeyTypeED25519, fedbox.KeyTypeRSA}),
			Value: fedbox.KeyTypeED25519,
		},
		&cli.StringFlag{
			Name:  "attributedTo",
			Usage: fmt.Sprintf("The IRI of the Actor we should use as author"),
		},
		&cli.StringSliceFlag{
			Name:  "tag",
			Usage: fmt.Sprintf("The tag(s) to attach to the actor"),
		},
	},
	Action: addActorAct(&ctl),
}

func addActorAct(ctl *Control) cli.ActionFunc {
	pauseFn := sendSignalToServerAct(ctl, syscall.SIGUSR1)
	return func(c *cli.Context) error {
		if err := pauseFn(c); err != nil {
			return errors.Annotatef(err, "Unable to pause server")
		}
		defer func() {
			if err := pauseFn(c); err != nil {
				ctl.Logger.WithContext(lw.Ctx{"err": err.Error()}).Warnf("Unable to pause server")
			}
		}()
		if err := ctl.Storage.Open(); err != nil {
			return errors.Annotatef(err, "Unable to open FedBOX storage for path %s", ctl.Conf.StoragePath)
		}
		defer ctl.Storage.Close()

		keyType := c.String("keyType")
		names := c.Args().Slice()
		if len(names) == 0 {
			name, err := loadFromStdin("Enter the actor's name")
			if err != nil {
				return errors.Errorf("Missing the actor's name")
			}
			names = append(names, string(name))
		}

		author := ap.Self(vocab.IRI(ctl.Conf.BaseURL))
		if authIRI := vocab.IRI(c.String("attributedTo")); len(authIRI) > 0 {
			act, err := ap.LoadActor(ctl.Storage, authIRI)
			if err != nil {
				return err
			}
			author = act
		}

		tags := make(vocab.ItemCollection, 0)

		objectsCollection := filters.ObjectsType.IRI(vocab.IRI(ctl.Conf.BaseURL))
		allObjects, _ := ctl.Storage.Load(objectsCollection)
		_ = vocab.OnCollectionIntf(allObjects, func(col vocab.CollectionInterface) error {
			for _, it := range col.Collection() {
				_ = vocab.OnObject(it, func(object *vocab.Object) error {
					for _, tag := range c.StringSlice("tag") {
						if object.Name.First().Value.String() != tag {
							continue
						}
						if object.AttributedTo.GetLink() != author.GetLink() {
							continue
						}
						_ = tags.Append(object)
					}
					return nil
				})
			}
			return nil
		})

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

			now := time.Now().UTC()
			p := &vocab.Person{
				Type: typ,
				// TODO(marius): when adding authentication to the command, we can set here the actor that executes it
				AttributedTo: author.GetLink(),
				Generator:    author.GetLink(),
				Published:    now,
				Summary: vocab.NaturalLanguageValues{
					{vocab.NilLangRef, vocab.Content("Generated actor")},
				},
				Updated: now,
				PreferredUsername: vocab.NaturalLanguageValues{
					{vocab.NilLangRef, vocab.Content(name)},
				},
			}
			if len(tags) > 0 {
				p.Tag = tags
			}
			if p, err = ctl.AddActor(p, pw, author); err != nil {
				//Errf("Error adding %s: %s\n", name, err)
				return err
			}
			fmt.Printf("Added %q [%s]: %s\n", typ, name, p.GetLink())
			if metaSaver, ok := ctl.Storage.(s.MetadataStorage); ok {
				if err := AddKeyToItem(metaSaver, p, keyType); err != nil {
					Errf("Error saving metadata for %s: %s", name, err)
				}
			}
			actors = append(actors, p)
		}
		return nil
	}
}

func wrapObjectInCreate(p vocab.Item, author vocab.Item) (vocab.Activity, error) {
	act := vocab.Activity{
		Type:    vocab.CreateType,
		To:      vocab.ItemCollection{vocab.PublicNS},
		Updated: time.Now().UTC(),
		Object:  p,
	}
	if act.AttributedTo == nil {
		act.AttributedTo = author.GetLink()
	}
	if act.Actor == nil {
		act.Actor = author
	}
	if !act.CC.Contains(author.GetLink()) {
		act.CC.Append(author.GetLink())
	}
	return act, nil
}

func (c *Control) AddObject(p *vocab.Object, author vocab.Actor) (*vocab.Object, error) {
	if c.Storage == nil {
		return nil, errors.Errorf("invalid storage backend")
	}
	if author.GetLink().Equals(auth.AnonymousActor.GetLink(), false) {
		self, err := ap.LoadActor(c.Storage, ap.DefaultServiceIRI(c.Conf.BaseURL))
		if err != nil {
			return nil, errors.NewNotFound(err, "unable to load current's instance Application actor")
		}
		if self.ID == "" {
			return nil, errors.NotFoundf("unable to load current's instance Application actor")
		}
		author = self
	}
	if author.GetID() == "" {
		return nil, errors.NotFoundf("unable to load current's instance Application actor: %s", c.Conf.BaseURL)
	}

	processor := saver(c)
	outbox := vocab.Outbox.Of(author).GetLink()
	if vocab.IsNil(outbox) {
		return nil, errors.Newf("unable to find Actor's outbox: %s", author)
	}

	create, err := wrapObjectInCreate(p, author)
	if err != nil {
		return nil, errors.Annotatef(err, "unable to wrap Object in Create activity")
	}
	if _, err = processor.ProcessClientActivity(create, author, outbox); err != nil {
		return nil, err
	}
	return p, nil
}

func saver(ctl *Control) processing.P {
	baseIRI := vocab.IRI(ctl.Conf.BaseURL)
	db := ctl.Storage
	l := ctl.Logger.WithContext(lw.Ctx{"log": "processing"})
	p := processing.New(
		processing.WithIRI(baseIRI),
		processing.WithStorage(db),
		processing.WithIDGenerator(fedbox.GenerateID(baseIRI)),
		processing.WithClient(c.New(
			c.WithLogger(l),
			c.SkipTLSValidation(!ctl.Conf.Env.IsProd()),
		)),
		processing.WithLocalIRIChecker(s.IsLocalIRI(db)),
		processing.WithLogger(l),
	)
	return p
}

func (c *Control) AddActor(p *vocab.Person, pw []byte, author vocab.Actor) (*vocab.Person, error) {
	if c.Storage == nil {
		return nil, errors.Errorf("invalid storage backend")
	}
	if author.GetLink().Equals(auth.AnonymousActor.GetLink(), false) {
		self, err := ap.LoadActor(c.Storage, ap.DefaultServiceIRI(c.Conf.BaseURL))
		if err != nil {
			return nil, errors.NewNotFound(err, "unable to load current's instance Application actor")
		}
		if self.ID == "" {
			return nil, errors.NotFoundf("unable to load current's instance Application actor")
		}
		author = self
	}
	if author.GetID() == "" {
		return nil, errors.NotFoundf("unable to load current's instance Application actor: %s", c.Conf.BaseURL)
	}

	create, err := wrapObjectInCreate(p, author)
	if err != nil {
		return nil, errors.Annotatef(err, "unable to wrap Actor in Create activity")
	}

	outbox := vocab.Outbox.Of(author)
	if vocab.IsNil(outbox) {
		return nil, errors.Newf("unable to find Actor's outbox: %s", author)
	}

	ap := saver(c)
	if _, err := ap.ProcessClientActivity(create, author, outbox.GetLink()); err != nil {
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
		if err := ctl.Storage.Open(); err != nil {
			return errors.Annotatef(err, "Unable to open FedBOX storage for path %s", ctl.Conf.StoragePath)
		}
		defer ctl.Storage.Close()

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

	ap := saver(c)
	if _, err := ap.ProcessClientActivity(d, self, vocab.Outbox.Of(d.Actor).GetLink()); err != nil {
		return err
	}

	_ = printItem(d, "text")
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
		return outJSON(os.Stdout)(it)
	}
	return outText(os.Stdout)(it)
}

func listObjectsAct(ctl *Control) cli.ActionFunc {
	pauseFn := sendSignalToServerAct(ctl, syscall.SIGUSR1)
	return func(c *cli.Context) error {
		if err := pauseFn(c); err != nil {
			return errors.Annotatef(err, "Unable to pause server")
		}
		defer func() {
			if err := pauseFn(c); err != nil {
				ctl.Logger.WithContext(lw.Ctx{"err": err.Error()}).Warnf("Unable to pause server")
			}
		}()
		if err := ctl.Storage.Open(); err != nil {
			return errors.Annotatef(err, "Unable to open FedBOX storage for path %s", ctl.Conf.StoragePath)
		}
		defer ctl.Storage.Close()

		typeFl := c.StringSlice("type")

		var paths vocab.IRIs
		if c.NArg() == 0 {
			paths = append(paths,
				filters.ObjectsType.IRI(vocab.IRI(ctl.Conf.BaseURL)),
				filters.ActorsType.IRI(vocab.IRI(ctl.Conf.BaseURL)),
				filters.ActivitiesType.IRI(vocab.IRI(ctl.Conf.BaseURL)),
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
			return vocab.ItemOrderTimestamp(all[i], all[j])
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
		ff, _ := filters.FromIRI(iri)
		ff = append(ff, filters.HasType(typeFilter...))

		col, err := c.Storage.Load(IRIWithFilters(iri, ByType(typeFilter...)), ff...)
		if err != nil {
			return items, err
		}
		_ = vocab.OnItem(col, func(it vocab.Item) error {
			if !vocab.IsNil(it) {
				items = append(items, it)
			}
			return nil
		})
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
			Usage:       fmt.Sprintf("The type of activitypub object(s) to create"),
			DefaultText: fmt.Sprintf("Valid values: %v", ValidGenericTypes),
		},
		&cli.StringFlag{
			Name:  "name",
			Usage: fmt.Sprintf("The name of the activitypub object(s) to create"),
		},
		&cli.StringFlag{
			Name:  "attributedTo",
			Usage: fmt.Sprintf("The IRI of the Actor we should use as author"),
		},
		&cli.StringSliceFlag{
			Name:  "tag",
			Usage: fmt.Sprintf("The tag(s) to attach to the object"),
		},
	},
	Action: addObjectAct(&ctl),
}

var validObjects = append(vocab.ObjectTypes, vocab.ObjectType, "")

func addObjectAct(ctl *Control) cli.ActionFunc {
	pauseFn := sendSignalToServerAct(ctl, syscall.SIGUSR1)
	return func(c *cli.Context) error {
		if err := pauseFn(c); err != nil {
			return errors.Annotatef(err, "Unable to pause server")
		}
		defer func() {
			if err := pauseFn(c); err != nil {
				ctl.Logger.WithContext(lw.Ctx{"err": err.Error()}).Warnf("Unable to pause server")
			}
		}()
		if err := ctl.Storage.Open(); err != nil {
			return errors.Annotatef(err, "Unable to open FedBOX storage for path %s", ctl.Conf.StoragePath)
		}
		defer ctl.Storage.Close()

		f := make(filters.Checks, 0)

		incType := c.String("type")
		if !validObjects.Contains(vocab.ActivityVocabularyType(incType)) {
			return errors.Errorf("This command only supports only object of types %v", vocab.ObjectTypes)
		} else {
			incType = string(vocab.NoteType)
		}
		prop := "title"
		if vocab.ActorTypes.Contains(vocab.ActivityVocabularyType(incType)) {
			prop = "name"
		}

		if typ := types(incType); typ != nil {
			f = append(f, typ)
		}

		incName := c.String("name")
		if len(incName) == 0 {
			if n, err := loadFromStdin("Enter the %s's %s", incType, prop); err == nil {
				incName = string(n)
			}
		}
		if n := names(incName); n != nil {
			f = append(f, n)
		}

		authIRI := vocab.IRI(c.String("attributedTo"))
		if len(authIRI) == 0 {
			authIRI = vocab.IRI(ctl.Conf.BaseURL)
		}
		author, err := ap.LoadActor(ctl.Storage, authIRI, f...)
		if err != nil {
			return err
		}

		now := time.Now().UTC()
		p := &vocab.Object{
			Type: vocab.ActivityVocabularyType(incType),
			// TODO(marius): when adding authentication to the command, we can set here the actor that executes it
			AttributedTo: author.GetLink(),
			CC:           vocab.ItemCollection{vocab.PublicNS},
			Published:    now,
			Name:         vocab.DefaultNaturalLanguageValue(incName),
		}

		if p, err = ctl.AddObject(p, author); err != nil {
			return errors.Annotatef(err, "Unable to save object")
		}
		fmt.Printf("Added %s [%s]: %s\n", incType, incName, p.GetLink())

		return nil
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
		if err := ctl.Storage.Open(); err != nil {
			return errors.Annotatef(err, "Unable to open FedBOX storage for path %s", ctl.Conf.StoragePath)
		}
		defer ctl.Storage.Close()

		baseIRI := ctl.Conf.BaseURL
		toReplace := c.String("base")
		files := c.Args().Slice()

		for _, name := range files {
			buf, err := os.ReadFile(name)
			if err != nil {
				Errf("Error %s", err)
				continue
			}
			if len(buf) == 0 {
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
			_ = vocab.OnCollectionIntf(col, func(c vocab.CollectionInterface) error {
				for _, it := range c.Collection() {
					typ := it.GetType()
					fmt.Printf("Saving %s\n", it.GetID())

					var err error
					if vocab.ActivityTypes.Contains(typ) || vocab.IntransitiveActivityTypes.Contains(typ) {
						_ = vocab.OnIntransitiveActivity(it, func(a *vocab.IntransitiveActivity) error {
							if a == nil {
								Errf("invalid activity, is nil: %s", it.GetLink())
								return nil
							}
							if a.Actor == nil {
								Errf("invalid activity, actor is nil: %s", it.GetLink())
								return nil
							}
							actor, err := vocab.ToActor(a.Actor)
							if err != nil {
								actor = &vocab.Actor{ID: a.Actor.GetLink()}
							}
							activityPub := saver(ctl)
							it, err = activityPub.ProcessClientActivity(it, *actor, vocab.Outbox.Of(a.Actor).GetLink())
							return err
						})
					} else {
						it, err = ctl.Storage.Save(it)
					}
					if err != nil {
						Errf("Unable to save %s %s: %s", it.GetType(), it.GetID(), err)
						continue
					}
					count++
				}
				return nil
			})
			tot := time.Now().Sub(start)
			fmt.Printf("Elapsed time:          %4s\n", tot)
			if count > 0 {
				perIt := time.Duration(int64(tot) / int64(count))
				fmt.Printf("Elapsed time per item: %4s\n", perIt)
			}
		}
		fmt.Printf("Import done!\n")
		return nil
	}
}

var exportCmd = &cli.Command{
	Name:    "export",
	Aliases: []string{"dump"},
	Usage:   "Exports ActivityPub objects",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "path",
			Usage: fmt.Sprintf("The path where to output the items, if absent it will be printed to stdout."),
		},
	},
	Action: exportPubObjects(&ctl),
}

func dumpAll(iri vocab.IRI, f ...filters.Check) (vocab.ItemCollection, error) {
	col := make(vocab.ItemCollection, 0)
	objects, err := ctl.Storage.Load(iri, f...)
	if err != nil {
		return col, err
	}
	if objects.IsCollection() {
		_ = vocab.OnCollectionIntf(objects, func(c vocab.CollectionInterface) error {
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
		if err := ctl.Storage.Open(); err != nil {
			return errors.Annotatef(err, "Unable to open FedBOX storage for path %s", ctl.Conf.StoragePath)
		}
		defer ctl.Storage.Close()

		baseURL := vocab.IRI(ctl.Conf.BaseURL)
		objects := make(vocab.ItemCollection, 0)
		allCollections := vocab.CollectionPaths{filters.ActivitiesType, filters.ActorsType, filters.ObjectsType}
		for _, col := range allCollections {
			dump, err := dumpAll(vocab.IRIf(baseURL, col))
			if err != nil {
				return err
			}
			objects = append(objects, dump...)
		}
		if len(objects) == 0 {
			return errors.Errorf("No objects to export")
		}
		sort.Slice(objects, func(i, j int) bool {
			return vocab.ItemOrderTimestamp(objects[i], objects[j])
		})
		where := os.Stdout
		if c.String("path") != "" {
			f, err := os.OpenFile(c.String("path"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
			if err != nil {
				return err
			}
			where = f
		}
		return outJSON(where)(objects)
	}
}

func showObjectAct(ctl *Control) cli.ActionFunc {
	pauseFn := sendSignalToServerAct(ctl, syscall.SIGUSR1)
	return func(c *cli.Context) error {
		if err := pauseFn(c); err != nil {
			return errors.Annotatef(err, "Unable to pause server")
		}
		defer func() {
			if err := pauseFn(c); err != nil {
				ctl.Logger.WithContext(lw.Ctx{"err": err.Error()}).Warnf("Unable to pause server")
			}
		}()
		if err := ctl.Storage.Open(); err != nil {
			return errors.Annotatef(err, "Unable to open FedBOX storage for path %s", ctl.Conf.StoragePath)
		}
		defer ctl.Storage.Close()

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
			_ = printItem(ob, c.String("output"))
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
		if err := ctl.Storage.Open(); err != nil {
			return errors.Annotatef(err, "Unable to open FedBOX storage for path %s", ctl.Conf.StoragePath)
		}
		defer ctl.Storage.Close()

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
		if err := ctl.Storage.Open(); err != nil {
			return errors.Annotatef(err, "Unable to open FedBOX storage for path %s", ctl.Conf.StoragePath)
		}
		defer ctl.Storage.Close()

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

var indexCmd = &cli.Command{
	Name:    "index",
	Aliases: []string{"reindex"},
	Usage:   "Reindex current storage ActivityPub objects",
	Action:  indexPubObjects(&ctl),
}

type reindexer interface {
	Reindex() error
}

func indexPubObjects(ctl *Control) cli.ActionFunc {
	return func(c *cli.Context) error {
		start := time.Now()

		indexer, ok := ctl.Storage.(reindexer)
		if !ok {
			_, _ = fmt.Fprintf(os.Stderr, "Current storage engine %T does not support reindexing\n", ctl.Storage)
			return errors.Newf("unsupported storage engine")
		}
		if err := ctl.Storage.Open(); err != nil {
			return errors.Annotatef(err, "Unable to open FedBOX storage for path %s", ctl.Conf.StoragePath)
		}
		defer ctl.Storage.Close()

		if err := indexer.Reindex(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Indexing failed: %s", err)
			return err
		}

		tot := time.Now().Sub(start)
		fmt.Printf("Elapsed time:          %s\n", tot)

		return nil
	}
}
