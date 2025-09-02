package cmd

import (
	"fmt"
	"os"
	"time"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox"
	"github.com/go-ap/fedbox/storage"
	"github.com/go-ap/jsonld"
)

type Accounts struct {
	Export  Export         `cmd:"" help:"Exports accounts metadata."`
	Import  Import         `cmd:"" help:"Imports accounts metadata."`
	GenKeys GenKeys        `cmd:"" help:"Generate public/private key pairs for actors that are missing them."`
	Pass    ChangePassword `cmd:"" help:"Change password for an actor."`
}

type Export struct{}

func (e Export) Run(ctl *Control) error {
	metaLoader, ok := ctl.Storage.(storage.MetadataStorage)
	if !ok {
		return errors.Newf("")
	}

	iri := SearchActorsIRI(vocab.IRI(ctl.Conf.BaseURL), ByType(vocab.PersonType))
	col, err := ctl.Storage.Load(iri)
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

	allMeta := make(map[vocab.IRI]auth.Metadata, len(items))
	for _, it := range items {
		if it.GetType() != vocab.PersonType {
			continue
		}
		m := new(auth.Metadata)
		if err = metaLoader.LoadMetadata(it.GetLink(), m); err != nil {
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

type Import struct {
	Files []*os.File `arg:""`
}

func (i Import) Run(ctl *Control) error {
	metaLoader, ok := ctl.Storage.(storage.MetadataStorage)
	if !ok {
		return errors.Newf("")
	}
	for _, f := range i.Files {
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
			Errf("Empty file %s", f.Name())
			continue
		}

		metadata := make(map[vocab.IRI]auth.Metadata)
		err = jsonld.Unmarshal(buf, &metadata)
		if err != nil {
			Errf("Error unmarshaling JSON: %s", err)
			continue
		}
		start := time.Now()
		count := 0
		for iri, m := range metadata {
			if err = metaLoader.SaveMetadata(iri, m); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "unable to save metadata for %s: %s", iri, err)
				continue
			}
			count++
		}

		tot := time.Now().Sub(start)
		fmt.Printf("Elapsed time:          %s\n", tot)
		if count > 0 {
			perIt := time.Duration(int64(tot) / int64(count))
			fmt.Printf("Elapsed time per item: %s\n", perIt)
		}
	}
	return nil
}

type GenKeys struct {
	Type string      `help:"Type of keys to generate." name:"key-type" enum:"${keyTypes}" default:"${defaultKeyType}"`
	IRIs []vocab.IRI `arg:"" optional:"" name:"iri" help:"Actors for which to generate the keys. Defaults to all actors if missing."`
}

func (g GenKeys) Run(ctl *Control) error {
	typ := g.Type
	metaSaver, ok := ctl.Storage.(storage.MetadataStorage)
	if !ok {
		return errors.Newf("storage doesn't support saving key")
	}

	col := make(vocab.ItemCollection, 0)
	for _, iri := range g.IRIs {
		actors, err := ctl.Storage.Load(iri)
		if err != nil {
			Errf(err.Error())
			continue
		}
		_ = vocab.OnActor(actors, func(act *vocab.Actor) error {
			col = append(col, act)
			return nil
		})
	}

	if len(g.IRIs) == 0 {
		// TODO(marius): we should improve this with filtering based on public key existing in the actor,
		//  and with batching.
		iri := SearchActorsIRI(vocab.IRI(ctl.Conf.BaseURL), ByType(vocab.PersonType))
		actors, err := ctl.Storage.Load(iri)
		if err != nil {
			return err
		}
		_ = vocab.OnCollectionIntf(actors, func(colIntf vocab.CollectionInterface) error {
			for _, it := range colIntf.Collection() {
				if !vocab.ActorTypes.Contains(it.GetType()) {
					continue
				}
				act, _ := vocab.ToActor(it)
				_ = col.Append(act)
			}
			return nil
		})
	}

	for _, it := range col {
		if !vocab.ActorTypes.Contains(it.GetType()) {
			continue
		}
		if err := fedbox.AddKeyToItem(metaSaver, it, typ); err != nil {
			Errf("Error: %s", err.Error())
		}
	}
	return nil
}

type ChangePassword struct {
	IRI vocab.IRI `arg:"" optional:"" name:"iri" help:"The actor for which to change the password."`
}

func (c ChangePassword) Run(ctl *Control) error {
	actors, err := ctl.Storage.Load(c.IRI)
	if err != nil {
		return err
	}
	actor, err := vocab.ToActor(actors)
	if err != nil {
		return err
	}
	pw, err := loadPwFromStdin(true, "%s's", nameOf(actor))
	if err != nil {
		return err
	}
	if pw == nil {
		return errors.Errorf("empty password")
	}

	pwManager, ok := ctl.Storage.(storage.PasswordChanger)
	if !ok {
		return errors.Errorf("unable to save password for current storage %T", ctl.Storage)
	}
	return pwManager.PasswordSet(c.IRI, pw)
}
