package cmd

import (
	"fmt"
	"os"
	"time"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/storage"
	"github.com/go-ap/jsonld"
	"github.com/urfave/cli/v2"
)

var AccountsCmd = &cli.Command{
	Name:  "accounts",
	Usage: "Accounts helper",
	Subcommands: []*cli.Command{
		exportAccountsMetadataCmd,
		importAccountsMetadataCmd,
		generateKeysCmd,
	},
}

var exportAccountsMetadataCmd = &cli.Command{
	Name:   "export",
	Usage:  "Exports accounts metadata",
	Action: exportAccountsMetadata(&ctl),
}

func exportAccountsMetadata(ctl *Control) cli.ActionFunc {
	return func(c *cli.Context) error {
		metaLoader, ok := ctl.Storage.(storage.MetadataTyper)
		if !ok {
			return errors.Newf("")
		}

		baseIRI := ap.ActorsType.IRI(vocab.IRI(ctl.Conf.BaseURL))
		f := ap.FiltersNew(
			ap.IRI(baseIRI),
			ap.Type(vocab.PersonType),
		)
		col, err := ctl.Storage.Load(f.GetLink())
		if err != nil {
			return err
		}

		items := make(vocab.ItemCollection, 0)
		if col.IsCollection() {
			err = vocab.OnCollectionIntf(col, func(c vocab.CollectionInterface) error {
				items = append(items, c.Collection()...)
				return nil
			})
			if err != nil {
				return err
			}
		} else {
			items = append(items, col)
		}

		allMeta := make(map[vocab.IRI]storage.Metadata, len(items))
		for _, it := range items {
			if it.GetType() != vocab.PersonType {
				continue
			}
			m, err := metaLoader.LoadMetadata(it.GetLink())
			if err != nil {
				//Errf("Error loading metadata for %s: %s", it.GetLink(), err)
				continue
			}
			if m == nil {
				//Errf("Error loading metadata for %s, nil metadata", it.GetLink())
				continue
			}
			allMeta[it.GetLink()] = *m
		}
		bytes, err := jsonld.Marshal(allMeta)
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", bytes)
		return nil
	}
}

var importAccountsMetadataCmd = &cli.Command{
	Name:   "import",
	Usage:  "Imports accounts metadata",
	Action: importAccountsMetadata(&ctl),
}

func importAccountsMetadata(ctl *Control) cli.ActionFunc {
	return func(c *cli.Context) error {
		files := c.Args().Slice()
		metaLoader, ok := ctl.Storage.(storage.MetadataTyper)
		if !ok {
			return errors.Newf("")
		}
		for _, name := range files {
			f, err := os.Open(name)
			if err != nil {
				if os.IsNotExist(err) {
					Errf("Invalid path %s", name)
				} else {
					Errf("Error %s", err)
				}
				continue
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

			metadata := make(map[vocab.IRI]storage.Metadata, 0)
			err = jsonld.Unmarshal(buf, &metadata)
			if err != nil {
				Errf("Error unmarshaling JSON: %s", err)
				continue
			}
			start := time.Now()
			count := 0
			for iri, m := range metadata {
				metaLoader.SaveMetadata(m, iri)
			}

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

var generateKeysCmd = &cli.Command{
	Name:      "gen-keys",
	Usage:     "Generate public/private key pairs for actors that are missing them",
	ArgsUsage: "IRI...",
	Action:    generateKeys(&ctl),
}

func AddKeyToItem(metaSaver storage.MetadataTyper, it vocab.Item) error {
	if err := vocab.OnActor(it, fedbox.AddKeyToPerson(metaSaver)); err != nil {
		return errors.Annotatef(err, "failed to process actor: %s", it.GetID())
	}
	if _, err := ctl.Storage.Save(it); err != nil {
		return errors.Annotatef(err, "failed to save actor: %s", it.GetID())
	}
	return nil
}

func generateKeys(ctl *Control) cli.ActionFunc {
	return func(c *cli.Context) error {
		metaSaver, ok := ctl.Storage.(storage.MetadataTyper)
		if !ok {
			return errors.Newf("storage doesn't support saving key")
		}

		col := make(vocab.ItemCollection, 0)
		for i := 0; i <= c.Args().Len(); i++ {
			iri := c.Args().Get(i)
			act, err := ctl.Storage.Load(vocab.IRI(iri))
			if err != nil {
				Errf(err.Error())
				continue
			}
			vocab.OnActor(act, func(ob *vocab.Actor) error {
				col = append(col, ob)
				return nil
			})
		}

		if c.Args().Len() == 0 {
			baseIRI := ap.ActorsType.IRI(vocab.IRI(ctl.Conf.BaseURL))
			filterFns := []ap.FilterFn{
				ap.IRI(baseIRI),
				ap.Type(vocab.PersonType),
			}
			f := ap.FiltersNew(filterFns...)
			// TODO(marius): we should improve this with filtering based on public key existing in the actor,
			//  and with batching.
			actors, err := ctl.Storage.Load(f.GetLink())
			if err != nil {
				return err
			}
			vocab.OnObject(actors, func(ob *vocab.Object) error {
				col = append(col, ob)
				return nil
			})
		}

		return vocab.OnCollectionIntf(col, func(c vocab.CollectionInterface) error {
			for _, it := range c.Collection() {
				if !vocab.ActorTypes.Contains(it.GetType()) {
					continue
				}
				if err := AddKeyToItem(metaSaver, it); err != nil {
					Errf("Error: %s", err.Error())
				}
			}
			return nil
		})
	}
}
