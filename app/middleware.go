package app

import (
	"context"
	"github.com/go-ap/activitypub/storage"
	local "github.com/go-ap/fedbox/internal/context"
	"github.com/go-ap/fedbox/internal/log"
	"net/http"
)

func Repo(loader storage.Loader) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			newCtx := context.WithValue(ctx, local.RepositoryKey, loader)
			next.ServeHTTP(w, r.WithContext(newCtx))
		}
		return http.HandlerFunc(fn)
	}
}

func ActorFromAuthHeader(next http.Handler) http.Handler {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := log.New()
		act, err := LoadActorFromAuthHeader(r, logger)
		if err != nil {
			if IsUnauthorized(err) {
				if challenge := Challenge(err); len(challenge) > 0 {
					w.Header().Add("WWW-Authenticate", challenge)
				}
			}
			logger.Warnf("%s", err)
		}
		if act != nil {
			ctx := r.Context()
			newCtx := context.WithValue(ctx, local.ActorKey, act)
			next.ServeHTTP(w, r.WithContext(newCtx))
		}
		next.ServeHTTP(w, r)
	})
	return http.HandlerFunc(fn)
}
