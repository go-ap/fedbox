package storage

import (
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/processing"
)

type CanBootstrap interface {
	CreateService(pub.Service) error
}

type PasswordChanger interface {
	PasswordSet(pub.Item, []byte) error
	PasswordCheck(pub.Item, []byte) error
}

type Metadata struct {
	Pw         []byte `jsonld:"pw,omitempty"`
	PrivateKey []byte `jsonld:"key,omitempty"`
}

type MetadataTyper interface {
	LoadMetadata(pub.IRI) (*Metadata, error)
	SaveMetadata(Metadata, pub.IRI) error
}

type MimeTypeSaver interface {
	SaveNaturalLanguageValues(pub.NaturalLanguageValues) error
	SaveMimeTypeContent(pub.MimeType, pub.NaturalLanguageValues) error
}

type Resetter interface {
	Reset()
}

type OptionFn func(s processing.Store) error
