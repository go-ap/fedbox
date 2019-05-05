package context

import (
	"context"
	"github.com/go-ap/activitypub/storage"
)
type CtxtKey string

var OwnerKey = CtxtKey("__owner")
var ActorKey = CtxtKey("__actor")
var RepositoryKey = CtxtKey("__repo")

func ActivityLoader(ctx context.Context) (storage.ActivityLoader, bool) {
	ctxVal := ctx.Value(RepositoryKey)
	s, ok := ctxVal.(storage.ActivityLoader)
	return s, ok
}

func ObjectLoader(ctx context.Context) (storage.ObjectLoader, bool) {
	ctxVal := ctx.Value(RepositoryKey)
	s, ok := ctxVal.(storage.ObjectLoader)
	return s, ok
}

func ActorLoader(ctx context.Context) (storage.ActorLoader, bool) {
	ctxVal := ctx.Value(RepositoryKey)
	s, ok := ctxVal.(storage.ActorLoader)
	return s, ok
}

func ActivitySaver(ctx context.Context) (storage.ActivitySaver, bool) {
	ctxVal := ctx.Value(RepositoryKey)
	s, ok := ctxVal.(storage.ActivitySaver)
	return s, ok
}

func ActorSaver(ctx context.Context) (storage.ActorSaver, bool) {
	ctxVal := ctx.Value(RepositoryKey)
	s, ok := ctxVal.(storage.ActorSaver)
	return s, ok
}

func ObjectSaver(ctx context.Context) (storage.ObjectSaver, bool) {
	ctxVal := ctx.Value(RepositoryKey)
	s, ok := ctxVal.(storage.ObjectSaver)
	return s, ok
}
