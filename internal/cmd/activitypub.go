package cmd

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"git.sr.ht/~mariusor/lw"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox"
	ap "github.com/go-ap/fedbox/activitypub"
	s "github.com/go-ap/fedbox/internal/storage"
	"github.com/go-ap/filters"
	"github.com/go-ap/processing"
)

type AddActorCmd struct {
	Type         vocab.ActivityVocabularyType `help:"The type of the ActivityPub actor to add."`
	KeyType      string                       `help:"Type of keys to generate: ${keyTypes}" enum:"${keyTypes}" default:"${defaultKeyType}"`
	AttributedTo vocab.IRI                    `help:"The IRI of the Actor we should use as author."`
	Tags         []string                     `name:"tag" help:"The tag(s) to attach to the actor."`
	Names        []string                     `arg:"" name:"name" help:"The name(s) of the actor."`
}

func (a AddActorCmd) Run(ctl *Control) error {
	keyType := a.KeyType
	if len(a.Names) == 0 {
		name, err := loadFromStdin("Enter the actor's name")
		if err != nil {
			return errors.Errorf("Missing the actor's name")
		}
		a.Names = append(a.Names, string(name))
	}

	author := ap.Self(vocab.IRI(ctl.Conf.BaseURL))
	if authIRI := a.AttributedTo; len(authIRI) > 0 {
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
				for _, tag := range a.Tags {
					if object.Name.First().String() != tag {
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
	for _, name := range a.Names {
		pw, err := loadPwFromStdin(true, "%s's", name)
		if err != nil {
			return err
		}
		if !vocab.ActorTypes.Contains(a.Type) {
			a.Type = vocab.PersonType
		}

		now := time.Now().UTC()
		p := &vocab.Person{
			Type: a.Type,
			// TODO(marius): when adding authentication to the command, we can set here the actor that executes it
			AttributedTo: author.GetLink(),
			Generator:    author.GetLink(),
			Published:    now,
			Summary: vocab.NaturalLanguageValues{
				vocab.NilLangRef: vocab.Content("Generated actor"),
			},
			Updated: now,
			PreferredUsername: vocab.NaturalLanguageValues{
				vocab.NilLangRef: vocab.Content(name),
			},
		}
		if len(tags) > 0 {
			p.Tag = tags
		}
		if p, err = ctl.AddActor(p, pw, author); err != nil {
			//Errf("Error adding %s: %s\n", name, err)
			return err
		}
		fmt.Printf("Added %q [%s]: %s\n", a.Type, name, p.GetLink())
		if metaSaver, ok := ctl.Storage.(fedbox.MetadataStorage); ok {
			if err := fedbox.AddKeyToItem(metaSaver, p, keyType); err != nil {
				Errf("Error saving metadata for %s: %s", name, err)
			}
		}
		actors = append(actors, p)
	}
	return nil
}

type ActorsCmd struct {
	Add AddActorCmd `cmd:"" help:"Adds an ActivityPub actor."`
}

type Pub struct {
	Actors ActorsCmd `cmd:"" name:"actor" help:"Actor management helper."`
	Add    AddCmd    `cmd:"" name:"add" help:"Adds a new object."`
	List   ListCmd   `cmd:"" help:"Lists objects."`
	Info   InfoCmd   `cmd:"" help:"Show information about an object."`
	Delete DeleteCmd `cmd:"" help:"Deletes an ActivityPub object."`
	Move   MoveCmd   `cmd:"" help:"Move ActivityPub objects to a new collection."`
	Copy   CopyCmd   `cmd:"" help:"Copy ActivityPub objects."`
	Index  IndexCmd  `cmd:"" help:"Reindex current storage ActivityPub objects."`
	Export ExportCmd `cmd:"" help:"Exports ActivityPub objects."`
	Import ImportCmd `cmd:"" help:"Imports ActivityPub objects."`
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
		_ = act.CC.Append(author.GetLink())
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
	cl := fedbox.Client(http.DefaultTransport, ctl.Conf, l)
	p := processing.New(
		processing.WithIRI(baseIRI), processing.WithLogger(l),
		processing.WithStorage(db), processing.WithClient(cl),
		processing.WithIDGenerator(fedbox.GenerateID(baseIRI)),
		processing.WithLocalIRIChecker(s.IsLocalIRI(db)),
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

	if _, err := saver(c).ProcessClientActivity(create, author, outbox.GetLink()); err != nil {
		return nil, err
	}

	if pwManager, ok := c.Storage.(fedbox.PasswordChanger); ok && pw != nil {
		err = pwManager.PasswordSet(p.GetLink(), pw)
	}
	return p, err
}

var ValidGenericTypes = vocab.ActivityVocabularyTypes{vocab.ObjectType, vocab.ActorType}

type DeleteCmd struct {
	Reason    string      `help:"The reason why we want to delete the item"`
	InReplyTo []string    `help:"If deletion is a followup on moderation activities"`
	IRIs      []vocab.IRI `arg:"" name:"iris"`
}

func (d DeleteCmd) Run(ctl *Control) error {
	return ctl.DeleteObjects(d.Reason, d.InReplyTo, d.IRIs...)
}

func (c *Control) DeleteObjects(reason string, inReplyTo []string, ids ...vocab.IRI) error {
	invalidRemoveTypes := append(append(vocab.ActivityTypes, vocab.IntransitiveActivityTypes...), vocab.TombstoneType)
	self := ap.Self(vocab.IRI(c.Conf.BaseURL))

	d := new(vocab.Delete)
	d.Type = vocab.DeleteType
	d.To = vocab.ItemCollection{vocab.PublicNS}
	d.CC = make(vocab.ItemCollection, 0)
	if reason != "" {
		d.Content = vocab.NaturalLanguageValuesNew()
		_ = d.Content.Append(vocab.NilLangRef, vocab.Content(reason))
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
	for _, iri := range ids {
		it, err := c.Storage.Load(iri)
		if err != nil {
			continue
		}
		// NOTE(marius): this should work if "it" is a collection or a single object
		_ = vocab.OnObject(it, func(o *vocab.Object) error {
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

	if _, err := saver(c).ProcessClientActivity(d, self, vocab.Outbox.Of(d.Actor).GetLink()); err != nil {
		return err
	}

	_ = printItem(d, "text")
	return nil
}

type ListCmd struct {
	Type   []vocab.ActivityVocabularyType `help:"The type of ActivityPub object to list" default:"${defaultObjectTypes}"`
	Output string                         `help:"The format in which to output the items." enum:"text,json" default:"text"`
	IRIs   []vocab.IRI                    `arg:"" name:"iris"`
}

func printItem(it vocab.Item, outType string) error {
	if outType == "json" {
		return outJSON(os.Stdout)(it)
	}
	return outText(os.Stdout)(it)
}

func (l ListCmd) Run(ctl *Control) error {
	typeFl := l.Type

	var paths vocab.IRIs
	if len(l.IRIs) == 0 {
		paths = append(paths,
			filters.ObjectsType.IRI(vocab.IRI(ctl.Conf.BaseURL)),
			filters.ActorsType.IRI(vocab.IRI(ctl.Conf.BaseURL)),
			filters.ActivitiesType.IRI(vocab.IRI(ctl.Conf.BaseURL)),
		)
	} else {
		for _, path := range l.IRIs {
			u, err := url.ParseRequestURI(path.String())
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
	_ = printItem(all, l.Output)
	return nil
}

func loadPubTypes(types ...vocab.ActivityVocabularyType) []vocab.ActivityVocabularyType {
	objectTyp := make(vocab.ActivityVocabularyTypes, 0)
	actorTyp := make(vocab.ActivityVocabularyTypes, 0)
	activityTyp := make(vocab.ActivityVocabularyTypes, 0)
	if len(types) == 0 {
		objectTyp = vocab.ObjectTypes
		actorTyp = vocab.ActorTypes
		activityTyp = vocab.ActivityTypes
	} else {
		for _, t := range types {
			if vocab.ObjectTypes.Contains(t) {
				objectTyp = append(objectTyp, t)
			}
			if vocab.ActorTypes.Contains(t) {
				actorTyp = append(actorTyp, t)
			}
			if vocab.ActivityTypes.Contains(t) {
				activityTyp = append(activityTyp, t)
			}
			if strings.ToLower(string(t)) == strings.ToLower(string(vocab.ObjectType)) {
				objectTyp = vocab.ObjectTypes
			}
			if strings.ToLower(string(t)) == strings.ToLower(string(vocab.ActorType)) {
				actorTyp = vocab.ActorTypes
			}
			if strings.ToLower(string(t)) == strings.ToLower(string(vocab.ActivityType)) {
				activityTyp = vocab.ActivityTypes
			}
		}
	}
	return append(append(objectTyp, actorTyp...), activityTyp...)
}

func (c *Control) List(iris vocab.IRIs, types ...vocab.ActivityVocabularyType) (vocab.ItemCollection, error) {
	var typeFilter []vocab.ActivityVocabularyType
	if len(types) > 0 {
		typeFilter = loadPubTypes(types...)
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

type AddCmd struct {
	Type         vocab.ActivityVocabularyType `help:"The type of ActivityPub object(s) to create." default:"${defaultObjectTypes}"`
	Name         string                       `help:"The name of the ActivityPub object(s) to create."`
	AttributedTo string                       `help:"The IRI of the Actor we should use as author"`
	Tag          []string                     `help:"The tag(s) to attach to the object."`
}

var validObjects = append(vocab.ObjectTypes, vocab.ObjectType, "")

func (a AddCmd) Run(ctl *Control) error {
	f := make(filters.Checks, 0)

	incType := a.Type
	if !validObjects.Contains(incType) {
		return errors.Errorf("This command only supports only object of types %v", vocab.ObjectTypes)
	} else {
		incType = vocab.NoteType
	}
	prop := "title"
	if vocab.ActorTypes.Contains(incType) {
		prop = "name"
	}

	if typ := types(incType); typ != nil {
		f = append(f, typ)
	}

	incName := a.Name
	if len(incName) == 0 {
		if n, err := loadFromStdin("Enter the %s's %s", incType, prop); err == nil {
			incName = string(n)
		}
	}
	if n := names(incName); n != nil {
		f = append(f, n)
	}

	authIRI := vocab.IRI(a.AttributedTo)
	if len(authIRI) == 0 {
		authIRI = vocab.IRI(ctl.Conf.BaseURL)
	}
	author, err := ap.LoadActor(ctl.Storage, authIRI, f...)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	p := &vocab.Object{
		Type: incType,
		// TODO(marius): when adding authentication to the command, we can set here the actor that executes it
		AttributedTo: author.GetLink(),
		CC:           vocab.ItemCollection{vocab.PublicNS},
		Published:    now,
		Name:         vocab.DefaultNaturalLanguage(incName),
	}

	if p, err = ctl.AddObject(p, author); err != nil {
		return errors.Annotatef(err, "Unable to save object")
	}
	fmt.Printf("Added %s [%s]: %s\n", incType, incName, p.GetLink())

	return nil
}

type ImportCmd struct {
	Base  vocab.IRI  `help:"The base IRI to replace"`
	Files []*os.File `arg:""`
}

func (i ImportCmd) Run(ctl *Control) error {
	baseIRI := ctl.Conf.BaseURL
	toReplace := i.Base

	for _, f := range i.Files {
		buf, err := io.ReadAll(f)
		if err != nil {
			Errf("Error %s", err)
			continue
		}
		if len(buf) == 0 {
			Errf("Empty file %s", f.Name())
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

type ExportCmd struct {
	File string `help:"The path where to output the items, if absent it will be printed to stdout."`
}

func dumpAll(ctl *Control, iri vocab.IRI, f ...filters.Check) (vocab.ItemCollection, error) {
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

func (e ExportCmd) Run(ctl *Control) error {
	baseURL := vocab.IRI(ctl.Conf.BaseURL)
	objects := make(vocab.ItemCollection, 0)
	allCollections := vocab.CollectionPaths{filters.ActivitiesType, filters.ActorsType, filters.ObjectsType}
	for _, col := range allCollections {
		dump, err := dumpAll(ctl, vocab.IRIf(baseURL, col))
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
	if e.File != "" {
		f, err := os.OpenFile(e.File, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			return err
		}
		where = f
	}
	return outJSON(where)(objects)
}

type InfoCmd struct {
	IRIs   []vocab.IRI `arg:"" name:"iris"`
	Output string      `help:"The format in which to output the items." enum:"text,json" default:"text"`
}

func (l InfoCmd) Run(ctl *Control) error {
	objects := make(vocab.ItemCollection, 0)
	if len(l.IRIs) == 0 {
		return errors.Errorf("No IRIs passed")
	}
	for _, iri := range l.IRIs {
		ob, err := ctl.Storage.Load(iri)
		if err != nil {
			Errf(err.Error())
			continue
		}
		objects = append(objects, ob)
	}

	for _, ob := range objects {
		_ = printItem(ob, l.Output)
	}
	return nil
}

func (c *Control) operateOnObjects(fn func(col vocab.IRI, it vocab.Item) error, to vocab.IRI, from ...vocab.IRI) error {
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
				return c.operateOnObjects(fn, to, col.Collection().IRIs()...)
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

type MoveCmd struct {
	To   vocab.IRI   `help:"Collection to which to move the objects." required:""`
	IRIs []vocab.IRI `arg:"" name:"iris"`
}

func (m MoveCmd) Run(ctl *Control) error {
	return ctl.MoveObjects(m.To, m.IRIs...)
}

func (c *Control) MoveObjects(to vocab.IRI, from ...vocab.IRI) error {
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

type CopyCmd struct {
	To   vocab.IRI   `help:"Collection to which to move the objects." required:""`
	IRIs []vocab.IRI `arg:"" name:"iris"`
}

func (c CopyCmd) Run(ctl *Control) error {
	return ctl.CopyObjects(c.To, c.IRIs...)
}

func (c *Control) CopyObjects(to vocab.IRI, from ...vocab.IRI) error {
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

type IndexCmd struct{}

type reindexer interface {
	Reindex() error
}

func (i IndexCmd) Run(ctl *Control) error {
	start := time.Now()

	indexer, ok := ctl.Storage.(reindexer)
	if !ok {
		_, _ = fmt.Fprintf(os.Stderr, "Current storage engine %T does not support reindexing\n", ctl.Storage)
		return errors.Newf("unsupported storage engine")
	}

	if err := indexer.Reindex(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Indexing failed: %s", err)
		return err
	}

	tot := time.Now().Sub(start)
	fmt.Printf("Elapsed time:          %s\n", tot)

	return nil
}
