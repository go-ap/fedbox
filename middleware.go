package fedbox

import (
	"net/http"
	"path"

	"github.com/go-chi/chi/v5"
)

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
