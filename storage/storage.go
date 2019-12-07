package storage

import pub "github.com/go-ap/activitypub"

type CanBootstrap interface {
	CreateService(pub.Service) error
}
