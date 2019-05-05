package app

import (
	"context"
	"github.com/go-ap/activitypub/storage"
)

func ContextActivityLoader(ctx context.Context) (storage.ActivityLoader, bool) {
	ctxVal := ctx.Value(RepositoryCtxtKey)
	s, ok := ctxVal.(storage.ActivityLoader)
	return s, ok
}

func ContextObjectLoader(ctx context.Context) (storage.ObjectLoader, bool) {
	ctxVal := ctx.Value(RepositoryCtxtKey)
	s, ok := ctxVal.(storage.ObjectLoader)
	return s, ok
}

func ContextActorLoader(ctx context.Context) (storage.ActorLoader, bool) {
	ctxVal := ctx.Value(RepositoryCtxtKey)
	s, ok := ctxVal.(storage.ActorLoader)
	return s, ok
}

func ContextActivitySaver(ctx context.Context) (storage.ActivitySaver, bool) {
	ctxVal := ctx.Value(RepositoryCtxtKey)
	s, ok := ctxVal.(storage.ActivitySaver)
	return s, ok
}

func ContextActorSaver(ctx context.Context) (storage.ActorSaver, bool) {
	ctxVal := ctx.Value(RepositoryCtxtKey)
	s, ok := ctxVal.(storage.ActorSaver)
	return s, ok
}

func ContextObjectSaver(ctx context.Context) (storage.ObjectSaver, bool) {
	ctxVal := ctx.Value(RepositoryCtxtKey)
	s, ok := ctxVal.(storage.ObjectSaver)
	return s, ok
}
