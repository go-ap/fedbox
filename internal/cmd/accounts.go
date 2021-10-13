package cmd

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/storage"
	"github.com/go-ap/jsonld"
	"github.com/urfave/cli/v2"
	"golang.org/x/crypto/ed25519"
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

		baseIRI := ap.ActorsType.IRI(pub.IRI(ctl.Conf.BaseURL))
		f := ap.FiltersNew(
			ap.IRI(baseIRI),
			ap.Type(pub.PersonType),
		)
		col, err := ctl.Storage.Load(f.GetLink())
		if err != nil {
			return err
		}

		items := make(pub.ItemCollection, 0)
		if col.IsCollection() {
			err = pub.OnCollectionIntf(col, func(c pub.CollectionInterface) error {
				items = append(items, c.Collection()...)
				return nil
			})
			if err != nil {
				return err
			}
		} else {
			items = append(items, col)
		}

		allMeta := make(map[pub.IRI]storage.Metadata, len(items))
		for _, it := range items {
			if it.GetType() != pub.PersonType {
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

			metadata := make(map[pub.IRI]storage.Metadata, 0)
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

func AddKeyToPerson(metaSaver storage.MetadataTyper) func(act *pub.Actor) error {
	return func(act *pub.Actor) error {
		if act.Type != pub.PersonType {
			return nil
		}

		m, err := metaSaver.LoadMetadata(act.ID)
		if err != nil && !errors.IsNotFound(err) {
			return errors.Annotatef(err, "failed loading metadata: %s", act.ID)
		}
		if m == nil {
			m = new(storage.Metadata)
		}
		var pubB, prvB pem.Block
		if m.PrivateKey == nil {
			pubB, prvB = GenerateECKeyPair()
			m.PrivateKey = pem.EncodeToMemory(&prvB)
			if err = metaSaver.SaveMetadata(*m, act.ID); err != nil {
				return errors.Annotatef(err, "failed saving metadata for actor: %s", act.ID)
			}
		} else {
			pubB = publicKeyFrom(m.PrivateKey)
		}
		if len(pubB.Bytes) > 0 {
			act.PublicKey = pub.PublicKey{
				ID:           pub.IRI(fmt.Sprintf("%s#main", act.ID)),
				Owner:        act.ID,
				PublicKeyPem: string(pem.EncodeToMemory(&pubB)),
			}
		}
		return nil
	}
}

func AddKeyToItem(metaSaver storage.MetadataTyper, it pub.Item) error {
	if err := pub.OnActor(it, AddKeyToPerson(metaSaver)); err != nil {
		return errors.Annotatef(err, "failed to process actor: %s", it.GetID())
	}
	if _, err := ctl.Storage.Save(it); err != nil {
		return errors.Annotatef(err, "failed to save actor: %s", it.GetID())
	}
	return nil
}

func generateKeys(ctl *Control) cli.ActionFunc {
	return func(c *cli.Context) error {
		baseIRI := ap.ActorsType.IRI(pub.IRI(ctl.Conf.BaseURL))

		filterFns := []ap.FilterFn{
			ap.IRI(baseIRI),
			ap.Type(pub.PersonType),
		}

		actors := make([]string, 0)
		for i := 0; i <= c.Args().Len(); i++ {
			iri := c.Args().Get(i)
			ob, err := ctl.Storage.Load(pub.IRI(iri))
			if err != nil {
				Errf(err.Error())
				continue
			}
			actors = append(actors, ob.GetLink().String())
		}
		if len(actors) > 0 {
			filterFns = append(filterFns, ap.ItemKey(actors...))
		}

		f := ap.FiltersNew(filterFns...)
		// TODO(marius): we should improve this with filtering based on public key existing in the actor,
		//  and with batching.
		col, err := ctl.Storage.Load(f.GetLink())
		if err != nil {
			return err
		}
		metaSaver, ok := ctl.Storage.(storage.MetadataTyper)
		if !ok {
			return errors.Newf("storage doesn't support saving key")
		}
		return pub.OnCollectionIntf(col, func(c pub.CollectionInterface) error {
			for _, it := range c.Collection() {
				if err = AddKeyToItem(metaSaver, it); err != nil {
					Errf("Error: %s", err.Error())
				}
			}
			return nil
		})
	}
}
func publicKeyFrom(prvBytes []byte) pem.Block {
	prv, _ := pem.Decode(prvBytes)
	var pubKey crypto.PublicKey
	if key, _ := x509.ParseECPrivateKey(prvBytes); key != nil {
		pubKey = key.PublicKey
	}
	if key, _ := x509.ParsePKCS8PrivateKey(prv.Bytes); pubKey == nil && key != nil {
		switch k := key.(type) {
		case *rsa.PrivateKey:
			pubKey = k.PublicKey
		case *ecdsa.PrivateKey:
			pubKey = k.PublicKey
		case ed25519.PrivateKey:
			pubKey = k.Public()
		}
	}
	pubEnc, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return pem.Block{}
	}
	return pem.Block{Type: "PUBLIC KEY", Bytes: pubEnc}
}

func GenerateECKeyPair() (pem.Block, pem.Block) {
	// TODO(marius): make this actually produce proper keys
	keyPub, keyPrv, _ := ed25519.GenerateKey(rand.Reader)

	pubEnc, err := x509.MarshalPKIXPublicKey(keyPub)
	if err != nil {
		panic(err)
	}
	prvEnc, err := x509.MarshalPKCS8PrivateKey(keyPrv)
	if err != nil {
		panic(err)
	}
	p := pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubEnc,
	}
	r := pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: prvEnc,
	}
	return p, r
}
