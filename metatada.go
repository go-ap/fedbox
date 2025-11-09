package fedbox

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"git.sr.ht/~mariusor/storage-all"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	"golang.org/x/crypto/ed25519"
)

const (
	KeyTypeECDSA   = "ECDSA"
	KeyTypeED25519 = "ED25519"
	KeyTypeRSA     = "RSA"
)

type MetadataStorage interface {
	LoadMetadata(vocab.IRI, any) error
	SaveMetadata(vocab.IRI, any) error
}

type PasswordChanger interface {
	PasswordSet(vocab.IRI, []byte) error
	PasswordCheck(vocab.IRI, []byte) error
}

func AddKeyToItem(metaSaver MetadataStorage, it vocab.Item, typ string) error {
	if err := vocab.OnActor(it, AddKeyToPerson(metaSaver, typ)); err != nil {
		return errors.Annotatef(err, "failed to process actor: %s", it.GetID())
	}
	st, ok := metaSaver.(storage.FullStorage)
	if !ok {
		return errors.Newf("invalid item store, failed to save actor: %s", it.GetID())
	}
	if _, err := st.Save(it); err != nil {
		return errors.Annotatef(err, "failed to save actor: %s", it.GetID())
	}
	return nil
}

func AddKeyToPerson(metaSaver storage.MetadataStorage, typ string) func(act *vocab.Actor) error {
	// TODO(marius): add a way to pass if we should overwrite the keys
	//  for now we'll assume that if we're calling this, we want to do it
	overwriteKeys := true
	return func(act *vocab.Actor) error {
		if !vocab.ActorTypes.Contains(act.Type) {
			return nil
		}

		m := new(auth.Metadata)
		_ = metaSaver.LoadMetadata(act.ID, m)
		var pubB, prvB pem.Block
		if m.PrivateKey == nil || overwriteKeys {
			if typ == KeyTypeED25519 {
				pubB, prvB = GenerateECKeyPair()
			} else {
				pubB, prvB = GenerateRSAKeyPair()
			}
			m.PrivateKey = pem.EncodeToMemory(&prvB)
			if err := metaSaver.SaveMetadata(act.ID, m); err != nil {
				return errors.Annotatef(err, "failed saving metadata for actor: %s", act.ID)
			}
		} else {
			pubB = publicKeyFrom(m.PrivateKey)
		}
		if len(pubB.Bytes) > 0 {
			act.PublicKey = vocab.PublicKey{
				ID:           vocab.IRI(fmt.Sprintf("%s#main", act.ID)),
				Owner:        act.ID,
				PublicKeyPem: string(pem.EncodeToMemory(&pubB)),
			}
		}
		return nil
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

func GenerateRSAKeyPair() (pem.Block, pem.Block) {
	keyPrv, _ := rsa.GenerateKey(rand.Reader, 2048)

	keyPub := keyPrv.PublicKey
	pubEnc, err := x509.MarshalPKIXPublicKey(&keyPub)
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
