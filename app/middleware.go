package app

import (
	"context"
	"github.com/go-ap/activitypub/storage"
	"net/http"
)

type CtxtKey string

var RepositoryCtxtKey = CtxtKey("__repo")

func Repo (loader storage.Loader) func (next http.Handler) http.Handler{
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			newCtx := context.WithValue(ctx, RepositoryCtxtKey, loader)
			next.ServeHTTP(w, r.WithContext(newCtx))
		}
		return http.HandlerFunc(fn)
	}
}

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
