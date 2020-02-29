package storage

import pub "github.com/go-ap/activitypub"

type CanBootstrap interface {
	CreateService(pub.Service) error
}

type PasswordChanger interface {
	PasswordSet(pub.Item, []byte) error
	PasswordCheck(pub.Item, []byte) error
}
