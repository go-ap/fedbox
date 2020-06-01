package cmd

import (
	"encoding/json"
	"fmt"
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/storage"
	"gopkg.in/urfave/cli.v2"
)

var AccountsCmd = &cli.Command{
	Name:  "accounts",
	Usage: "Accounts helper",
	Subcommands: []*cli.Command{
		exportAccountsMetadataCmd,
		importAccountsMetadataCmd,
	},
}

var exportAccountsMetadataCmd = &cli.Command{
	Name:   "export",
	Usage:  "Exports accounts metadata",
	Action: exportAccountsMetadata(&ctl),
}

func exportAccountsMetadata(ctl *Control) cli.ActionFunc {
	return func(c *cli.Context) error {
		baseIRI := ap.ActorsType.IRI(pub.IRI(ctl.BaseURL))
		f := ap.FiltersNew(
			ap.IRI(baseIRI),
			ap.Type(pub.PersonType),
		)
		col, err := ctl.Storage.LoadCollection(f)
		if err != nil {
			return err
		}

		items := make(pub.ItemCollection, 0)
		err = pub.OnCollectionIntf(col, func(c pub.CollectionInterface) error {
			for _, tt := range c.Collection() {
				items = append(items, tt)
			}
			return nil
		})

		metaLoader, ok := ctl.Storage.(storage.MetadataTyper)
		if !ok {
			return errors.Newf("")
		}

		allMeta := make(map[pub.IRI]storage.Metadata, len(items))
		for _, it := range items {
			if it.GetType() == pub.PersonType {
				m, err := metaLoader.LoadMetadata(it.GetLink())
				if err != nil {
					Errf("Error loading metadata for %s: %s", it.GetLink(), err)
					continue
				}
				if m == nil {
					Errf("Error loading metadata for %s, nil metadata", it.GetLink())
					continue
				}
				allMeta[it.GetLink()] = *m
			}
		}
		bytes, err := json.Marshal(allMeta)
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
		return nil
	}
}
