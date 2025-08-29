package fedbox

import (
	"net/http"
	"path"

	"github.com/go-ap/errors"
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

var (
	errShuttingDown = errors.ServiceUnavailablef("server is shutting down")
	errOutOfOrder   = errors.ServiceUnavailablef("temporarily out of order")
)

func OutOfOrderMw(f *FedBOX) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			maybeOoOHandler := next
			if f.shuttingDown {
				maybeOoOHandler = errors.HandleError(errShuttingDown)
			} else if f.maintenanceMode {
				maybeOoOHandler = errors.HandleError(errOutOfOrder)
			}
			maybeOoOHandler.ServeHTTP(w, r)
		})
	}
}
