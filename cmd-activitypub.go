package fedbox

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"os"
	"sort"
	"time"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/filters"
)

type AddActorCmd struct {
	Type         vocab.ActivityVocabularyType `help:"The type of the ActivityPub actor to add."`
	KeyType      string                       `help:"Type of keys to generate: ${keyTypes}" enum:"${keyTypes}" default:"${defaultKeyType}"`
	AttributedTo vocab.IRI                    `help:"The IRI of the Actor we should use as author."`
	Tags         []string                     `name:"tag" help:"The tag(s) to attach to the actor."`
	Names        []string                     `arg:"" name:"name" help:"The name(s) of the actor."`
}

func (a AddActorCmd) Run(ctl *Base) error {
	keyType := a.KeyType
	if len(a.Names) == 0 {
		// FIXME(marius): pass stdIn also to the Run
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

	rw := muxReadWriter{Reader: ctl.in, Writer: ctl.out}
	var actors = make(vocab.ItemCollection, 0)
	for _, name := range a.Names {
		pw, err := loadPwFromStdin(rw, fmt.Sprintf("%s's password: ", name))
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
		_, _ = fmt.Fprintf(ctl.out, "Added %q [%s]: %s\n", a.Type, name, p.GetLink())
		pair, _ := ap.GenerateKeyPair(ap.KeyType(keyType))
		if pair != nil {
			if err := ap.AddKeyToItem(ctl.Storage, p, *pair); err != nil {
				Errf(ctl.err, "Error saving metadata for %s: %s", name, err)
			}
		}
		if pw != nil {
			err = ctl.Storage.PasswordSet(p.ID, pw)
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

var ValidGenericTypes = vocab.ActivityVocabularyTypes{vocab.ObjectType, vocab.ActorType}

type DeleteCmd struct {
	Reason    string      `help:"The reason why we want to delete the item"`
	InReplyTo []string    `help:"If deletion is a followup on moderation activities"`
	IRIs      []vocab.IRI `arg:"" name:"iris"`
}

func (d DeleteCmd) Run(ctl *Base) error {
	return ctl.DeleteObjects(d.Reason, d.InReplyTo, d.IRIs...)
}

type ListCmd struct {
	Type   []vocab.ActivityVocabularyType `help:"The type of ActivityPub object to list" default:"${defaultObjectTypes}"`
	Output string                         `help:"The format in which to output the items." enum:"text,json" default:"text"`
	IRIs   []vocab.IRI                    `arg:"" name:"iris"`
}

func (l ListCmd) Run(ctl *Base) error {
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
	_ = printItem(ctl.out, all, l.Output)
	return nil
}

type AddCmd struct {
	Type         vocab.ActivityVocabularyType `help:"The type of ActivityPub object(s) to create." default:"${defaultObjectTypes}"`
	Name         string                       `help:"The name of the ActivityPub object(s) to create."`
	AttributedTo string                       `help:"The IRI of the Actor we should use as author"`
	Tag          []string                     `help:"The tag(s) to attach to the object."`
}

var validObjects = append(vocab.ObjectTypes, vocab.ObjectType, "")

func (a AddCmd) Run(ctl *Base) error {
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
	_, _ = fmt.Fprintf(ctl.out, "Added %s [%s]: %s\n", incType, incName, p.GetLink())

	return nil
}

type ImportCmd struct {
	Base  vocab.IRI  `help:"The base IRI to replace"`
	Files []*os.File `arg:""`
}

func (i ImportCmd) Run(ctl *Base) error {
	baseIRI := ctl.Conf.BaseURL
	toReplace := i.Base

	for _, f := range i.Files {
		buf, err := io.ReadAll(f)
		if err != nil {
			Errf(ctl.err, "Error %s", err)
			continue
		}
		if len(buf) == 0 {
			Errf(ctl.err, "Empty file %s", f.Name())
			continue
		}

		if len(toReplace) > 0 {
			buf = bytes.Replace(buf, []byte(toReplace), []byte(baseIRI), -1)
		}
		ob, err := vocab.UnmarshalJSON(buf)
		if err != nil {
			Errf(ctl.err, "Error unmarshaling JSON: %s", err)
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
							Errf(ctl.err, "invalid activity, is nil: %s", it.GetLink())
							return nil
						}
						if a.Actor == nil {
							Errf(ctl.err, "invalid activity, actor is nil: %s", it.GetLink())
							return nil
						}
						actor, err := vocab.ToActor(a.Actor)
						if err != nil {
							actor = &vocab.Actor{ID: a.Actor.GetLink()}
						}
						activityPub := ctl.Saver(&ctl.Service)
						it, err = activityPub.ProcessClientActivity(it, *actor, vocab.Outbox.Of(a.Actor).GetLink())
						return err
					})
				} else {
					it, err = ctl.Storage.Save(it)
				}
				if err != nil {
					Errf(ctl.err, "Unable to save %s %s: %s", it.GetType(), it.GetID(), err)
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

func dumpAll(ctl *Base, iri vocab.IRI, f ...filters.Check) (vocab.ItemCollection, error) {
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

func (e ExportCmd) Run(ctl *Base) error {
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
		defer f.Close()
		where = f
	}
	return OutJSON(where)(objects)
}

type InfoCmd struct {
	IRIs   []vocab.IRI `arg:"" name:"iris"`
	Output string      `help:"The format in which to output the items." enum:"text,json" default:"text"`
}

func (l InfoCmd) Run(ctl *Base) error {
	objects := make(vocab.ItemCollection, 0)
	if len(l.IRIs) == 0 {
		return errors.Errorf("No IRIs passed")
	}
	for _, iri := range l.IRIs {
		ob, err := ctl.Storage.Load(iri)
		if err != nil {
			Errf(ctl.err, err.Error())
			continue
		}
		objects = append(objects, ob)
	}

	for _, ob := range objects {
		_ = printItem(ctl.out, ob, l.Output)
	}
	return nil
}

type MoveCmd struct {
	To   vocab.IRI   `help:"Collection to which to move the objects." required:""`
	IRIs []vocab.IRI `arg:"" name:"iris"`
}

func (m MoveCmd) Run(ctl *Base) error {
	return ctl.MoveObjects(m.To, m.IRIs...)
}

type CopyCmd struct {
	To   vocab.IRI   `help:"Collection to which to move the objects." required:""`
	IRIs []vocab.IRI `arg:"" name:"iris"`
}

func (c CopyCmd) Run(ctl *Base) error {
	return ctl.CopyObjects(c.To, c.IRIs...)
}

type IndexCmd struct{}

type reindexer interface {
	Reindex() error
}

func (i IndexCmd) Run(ctl *Base) error {
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
