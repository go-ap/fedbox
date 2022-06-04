package storage

import (
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/processing"
)

type CanBootstrap interface {
	CreateService(vocab.Service) error
}

type PasswordChanger interface {
	PasswordSet(vocab.Item, []byte) error
	PasswordCheck(vocab.Item, []byte) error
}

type Metadata struct {
	Pw         []byte `jsonld:"pw,omitempty"`
	PrivateKey []byte `jsonld:"key,omitempty"`
}

type MetadataTyper interface {
	LoadMetadata(vocab.IRI) (*Metadata, error)
	SaveMetadata(Metadata, vocab.IRI) error
}

type MimeTypeSaver interface {
	SaveNaturalLanguageValues(vocab.NaturalLanguageValues) error
	SaveMimeTypeContent(vocab.MimeType, vocab.NaturalLanguageValues) error
}

type Resetter interface {
	Reset()
}

type OptionFn func(s processing.Store) error
