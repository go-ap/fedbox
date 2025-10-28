package storage

import (
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/processing"
	"github.com/openshift/osin"
)

type clientSaver interface {
	// UpdateClient updates the client (identified by its id) and replaces the values with the values of client.
	UpdateClient(c osin.Client) error
	// CreateClient stores the client in the database and returns an error, if something went wrong.
	CreateClient(c osin.Client) error
	// RemoveClient removes a client (identified by id) from the database. Returns an error if something went wrong.
	RemoveClient(id string) error
}

type clientLister interface {
	// ListClients lists existing clients
	ListClients() ([]osin.Client, error)
	GetClient(id string) (osin.Client, error)
}

type FullStorage interface {
	Open() error
	clientSaver
	clientLister
	processing.Store
	processing.KeyLoader
	MetadataStorage
	PasswordChanger
	osin.Storage
}

type CanBootstrap interface {
	CreateService(vocab.Service) error
}

type PasswordChanger interface {
	PasswordSet(vocab.IRI, []byte) error
	PasswordCheck(vocab.IRI, []byte) error
}

type MetadataStorage interface {
	LoadMetadata(vocab.IRI, any) error
	SaveMetadata(vocab.IRI, any) error
}

type MimeTypeSaver interface {
	SaveNaturalLanguageValues(vocab.NaturalLanguageValues) error
	SaveMimeTypeContent(vocab.MimeType, vocab.NaturalLanguageValues) error
}

type Resetter interface {
	Reset()
}

type IRIChecker interface {
	IsLocalIRI(i vocab.IRI) bool
}

func IsLocalIRI(s processing.Store) processing.IRIValidator {
	if c, ok := s.(IRIChecker); ok {
		return c.IsLocalIRI
	}
	return func(i vocab.IRI) bool {
		return false
	}
}

type OptionFn func(s processing.Store) error
