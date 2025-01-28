package cmd

import (
	"errors"
	"fmt"
	"os"
	"time"

	"git.sr.ht/~mariusor/lw"
	vocab "github.com/go-ap/activitypub"
	http "github.com/go-ap/errors"
	"github.com/go-ap/fedbox"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/storage"
	s "github.com/go-ap/fedbox/storage"
	"github.com/go-ap/processing"
	"github.com/urfave/cli/v2"
)

var BootstrapCmd = &cli.Command{
	Name:  "bootstrap",
	Usage: "Bootstrap a new postgres or bolt database helper",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "root",
			Usage: "root account of postgres server (default: postgres)",
			Value: "postgres",
		},
		&cli.StringFlag{
			Name:  "sql",
			Usage: "path to the queries for initializing the database",
			Value: "postgres",
		},
		&cli.StringFlag{
			Name:  "key-type",
			Usage: fmt.Sprintf("Type of keys to generate: %v", []string{fedbox.KeyTypeED25519, fedbox.KeyTypeRSA}),
			Value: fedbox.KeyTypeED25519,
		},
	},
	Action: bootstrapAct(&ctl),
	Subcommands: []*cli.Command{
		reset,
	},
}

var reset = &cli.Command{
	Name:   "reset",
	Usage:  "reset an existing database",
	Action: resetAct(&ctl),
}

func resetAct(c *Control) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		err := Reset(c.Conf)
		if err != nil {
			return err
		}
		return Bootstrap(c.Conf, c.Service)
	}
}

func bootstrapAct(c *Control) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		keyType := ctx.String("keyType")
		c.Service = ap.Self(ap.DefaultServiceIRI(c.Conf.BaseURL))
		if err := Bootstrap(c.Conf, c.Service); err != nil {
			Errf("Error adding service: %s\n", err)
			return err
		}
		if metaSaver, ok := ctl.Storage.(s.MetadataTyper); ok {
			if err := AddKeyToItem(metaSaver, &c.Service, keyType); err != nil {
				Errf("Error saving metadata for service: %s", err)
				return err
			}
		}
		return nil
	}
}

func Bootstrap(conf config.Options, service vocab.Item) error {
	l := lw.Dev(lw.SetOutput(os.Stderr))
	if err := storage.BootstrapFn(conf); err != nil {
		return http.Annotatef(err, "Unable to create %s path for storage %s", conf.BaseStoragePath(), conf.Storage)
	}
	l.Infof("Successfully created %s db for storage %s", conf.BaseStoragePath(), conf.Storage)

	db, err := fedbox.Storage(conf, l)
	if err != nil {
		return http.Annotatef(err, "Unable to load FedBOX storage for path %s", conf.StoragePath)
	}
	if err = CreateService(db, service); err != nil {
		return http.Annotatef(err, "Unable to create FedBOX service %s for storage %s", service.GetID(), conf.Storage)
	}
	l.Infof("Successfully created FedBOX service %s for storage %s", service.GetID(), conf.Storage)
	return nil
}

func Reset(conf config.Options) error {
	l := lw.Dev(lw.SetOutput(os.Stderr))
	if err := storage.CleanFn(conf); err != nil {
		return http.Annotatef(err, "Unable to reset %s db for storage %s", conf.BaseStoragePath(), conf.Storage)
	}
	l.Infof("Successfully reset %s db for storage %s", conf.BaseStoragePath(), conf.Storage)
	return nil
}

func CreateService(r s.FullStorage, self vocab.Item) (err error) {
	_ = vocab.OnActor(self, func(service *vocab.Actor) error {
		service.Published = time.Now().UTC()
		return nil
	})
	self, err = r.Save(self)
	if err != nil {
		return err
	}

	rr, ok := r.(processing.CollectionStore)
	if !ok {
		return nil
	}

	col := func(iri vocab.IRI) vocab.CollectionInterface {
		return &vocab.OrderedCollection{
			ID:           iri,
			Type:         vocab.OrderedCollectionType,
			Published:    time.Now().UTC(),
			AttributedTo: self.GetLink(),
		}
	}
	return vocab.OnActor(self, func(service *vocab.Actor) error {
		var multi error
		for _, stream := range service.Streams {
			if _, err := rr.Create(col(stream.GetID())); err != nil {
				multi = errors.Join(multi, err)
			}
		}
		return multi
	})
}
