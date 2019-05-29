package app

import (
	"context"
	"github.com/go-ap/activitypub/storage"
	st "github.com/go-ap/fedbox/storage"
)

type CtxtKey string

var OwnerKey = CtxtKey("__owner")
var ActorKey = CtxtKey("__actor")
var RepositoryKey = CtxtKey("__repo")

func Loader(ctx context.Context) (st.Loader, bool) {
	ctxVal := ctx.Value(RepositoryKey)
	s, ok := ctxVal.(st.Loader)
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
