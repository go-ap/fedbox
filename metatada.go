package fedbox

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/storage"
	"golang.org/x/crypto/ed25519"
)

const (
	KeyTypeECDSA   = "ECDSA"
	KeyTypeED25519 = "ED25519"
	KeyTypeRSA     = "RSA"
)

func AddKeyToPerson(metaSaver storage.MetadataTyper, typ string) func(act *vocab.Actor) error {
	// TODO(marius): add a way to pass if we should overwrite the keys
	//  for now we'll assume that if we're calling this, we want to do it
	overwriteKeys := true
	return func(act *vocab.Actor) error {
		if !vocab.ActorTypes.Contains(act.Type) {
			return nil
		}

		m, _ := metaSaver.LoadMetadata(act.ID)
		if m == nil {
			m = new(auth.Metadata)
		}
		var pubB, prvB pem.Block
		if m.PrivateKey == nil || overwriteKeys {
			if typ == KeyTypeED25519 {
				pubB, prvB = GenerateECKeyPair()
			} else {
				pubB, prvB = GenerateRSAKeyPair()
			}
			m.PrivateKey = pem.EncodeToMemory(&prvB)
			if err := metaSaver.SaveMetadata(*m, act.ID); err != nil {
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
