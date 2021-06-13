package app

import (
	"context"
	"net/http"
	"path"

	"github.com/go-ap/auth"
	"github.com/go-ap/handlers"
	"github.com/go-ap/storage"
	"github.com/go-chi/chi"
)

// RepoMw adds an implementation of the storage.Loader to a Request's context so it can be used
// further in the middleware chain
func RepoMw(loader storage.ReadStore) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			newCtx := context.WithValue(ctx, handlers.RepositoryKey, loader)
			next.ServeHTTP(w, r.WithContext(newCtx))
		}
		return http.HandlerFunc(fn)
	}
}

// ActorFromAuthHeader tries to load a local actor from the OAuth2 or HTTP Signatures Authorization headers
func (f FedBOX) ActorFromAuthHeader(next http.Handler) http.Handler {
	// TODO(marius): move this to the auth package and also add the possibility of getting the logger as a parameter
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		act, err := f.OAuth.auth.LoadActorFromAuthHeader(r)
		if err != nil {
			// FIXME(marius): This needs to be moved to someplace where we specifically require authorization
			//    it should not trigger for every request like it does if it remains here.
			//if errors.IsUnauthorized(err) {
			//	if challenge := errors.Challenge(err); len(challenge) > 0 {
			//		w.Header().Add("WWW-Authenticate", challenge)
			//	}
			//}
			f.errFn("%s", err)
		}
		if id := act.GetID(); id.IsValid() {
			r = r.WithContext(context.WithValue(r.Context(), auth.ActorKey, act))
		}
		next.ServeHTTP(w, r)
	})
}

func CleanRequestPath(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())

		routePath := rctx.RoutePath
		if routePath == "" {
			if r.URL.RawPath != "" {
				routePath = r.URL.RawPath
			} else {
				routePath = r.URL.Path
			}
		}
		rctx.RoutePath = path.Clean(routePath)

		next.ServeHTTP(w, r)
	})
}
