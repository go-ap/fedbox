package ap

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
	"golang.org/x/crypto/ssh"
)

type KeyType string

const (
	KeyTypeECDSA   KeyType = "ECDSA"
	KeyTypeED25519 KeyType = "ED25519"
	KeyTypeRSA     KeyType = "RSA"
)

type MetadataStorage interface {
	LoadMetadata(vocab.IRI, any) error
	SaveMetadata(vocab.IRI, any) error
}

func AddKeyToItem(metaSaver MetadataStorage, it vocab.Item, pair KeyPair) error {
	if err := vocab.OnActor(it, AddKeyToPerson(metaSaver, pair)); err != nil {
		return errors.Annotatef(err, "failed to process actor: %s", it.GetID())
	}
	return nil
}

type KeyPair struct {
	Private crypto.PrivateKey
	Public  crypto.PublicKey
	Type    KeyType
}

func AddKeyToPerson(metaSaver storage.MetadataStorage, pair KeyPair) func(act *vocab.Actor) error {
	return func(act *vocab.Actor) error {
		if !vocab.ActorTypes.Contains(act.Type) {
			return nil
		}

		m := new(auth.Metadata)
		_ = metaSaver.LoadMetadata(act.ID, m)
		pubB, prvB, err := EncodeKeyPair(pair)
		if err != nil {
			return err
		}

		m.PrivateKey = pem.EncodeToMemory(&prvB)
		if err := metaSaver.SaveMetadata(act.ID, m); err != nil {
			return errors.Annotatef(err, "failed saving metadata for actor: %s", act.ID)
		}
		act.PublicKey.ID = vocab.IRI(fmt.Sprintf("%s#main", act.ID))
		act.PublicKey.Owner = act.ID
		act.PublicKey.PublicKeyPem = string(pem.EncodeToMemory(&pubB))

		if st, ok := metaSaver.(storage.FullStorage); ok {
			if _, err := st.Save(act); err != nil {
				return errors.Annotatef(err, "failed to save actor: %s", act.ID)
			}
		}
		return nil
	}
}

func KeyPairFromPrivateBytes(prvBytes []byte) (*KeyPair, error) {
	pair := new(KeyPair)
	key, err := ssh.ParseRawPrivateKey(prvBytes)
	if err != nil {
		return nil, err
	}
	switch k := key.(type) {
	case *rsa.PrivateKey:
		pair.Private = k
		pair.Public = &k.PublicKey
		pair.Type = KeyTypeRSA
	case *ecdsa.PrivateKey:
		pair.Private = k
		pair.Public = &k.PublicKey
		pair.Type = KeyTypeECDSA
	case *ed25519.PrivateKey:
		pair.Private = *k
		pair.Public = k.Public()
		pair.Type = KeyTypeED25519
	case ed25519.PrivateKey:
		pair.Private = k
		pair.Public = k.Public()
		pair.Type = KeyTypeED25519
	}
	return pair, nil
}

func GenerateKeyPair(typ KeyType) (*KeyPair, error) {
	var pub crypto.PublicKey
	var prv crypto.PrivateKey
	var err error
	if typ == KeyTypeED25519 {
		pub, prv, err = ed25519.GenerateKey(rand.Reader)
	} else {
		var rsaPrv *rsa.PrivateKey
		rsaPrv, err = rsa.GenerateKey(rand.Reader, 2048)
		if rsaPrv != nil {
			prv = rsaPrv
			pub = rsaPrv.Public()
		}
	}
	if err != nil {
		return nil, err
	}
	return &KeyPair{Private: prv, Public: pub, Type: KeyType(typ)}, nil
}

func EncodeKeyPair(p KeyPair) (pem.Block, pem.Block, error) {
	pubEnc, err := x509.MarshalPKIXPublicKey(p.Public)
	if err != nil {
		return pem.Block{}, pem.Block{}, err
	}
	prvEnc, err := x509.MarshalPKCS8PrivateKey(p.Private)
	if err != nil {
		return pem.Block{}, pem.Block{}, err
	}
	pub := pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubEnc,
	}
	prv := pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: prvEnc,
	}
	return pub, prv, nil
}
