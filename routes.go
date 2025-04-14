package fedbox

import (
	"net/http"

	"git.sr.ht/~mariusor/lw"
	"github.com/go-ap/errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (f *FedBOX) Routes() func(chi.Router) {
	return func(r chi.Router) {
		r.Use(middleware.RequestID)
		r.Use(middleware.RealIP)
		r.Use(CleanRequestPath)
		r.Use(SetCORSHeaders)
		r.Use(lw.Middlewares(f.logger)...)

		r.Method(http.MethodGet, "/", HandleItem(f))
		r.Method(http.MethodHead, "/", HandleItem(f))
		// TODO(marius): we can separate here the FedBOX specific collections from the ActivityPub spec ones
		//   using some regular expressions
		//   Eg: "/{collection:(inbox|outbox|followed)}"
		//   Eg: "/{collection:(activities|objects|actors|moderators|ignored|blocked|flagged)}"
		r.Route("/{collection}", f.CollectionRoutes(true))

		if f.conf.Env.IsDev() {
			r.Mount("/debug", middleware.Profiler())
		}

		r.Handle("/favicon.ico", errors.NotFound)
		r.NotFound(errors.NotFound.ServeHTTP)
		r.MethodNotAllowed(errors.HandleError(errors.MethodNotAllowedf("method not allowed")).ServeHTTP)
	}
}

func (f *FedBOX) CollectionRoutes(descend bool) func(chi.Router) {
	return func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Method(http.MethodGet, "/", HandleCollection(f))
			r.Method(http.MethodHead, "/", HandleCollection(f))
			r.Method(http.MethodPost, "/", HandleActivity(f))

			r.Route("/{id}", func(r chi.Router) {
				r.Method(http.MethodGet, "/", HandleItem(f))
				r.Method(http.MethodHead, "/", HandleItem(f))
				if descend {
					r.Route("/{collection}", f.CollectionRoutes(false))
				}
			})
		})
	}
}

func SetCORSHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
