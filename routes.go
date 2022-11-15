package fedbox

import (
	"net/http"

	"github.com/go-ap/errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (f FedBOX) CollectionRoutes(descend bool) func(chi.Router) {
	return func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Method(http.MethodGet, "/", HandleCollection(f))
			r.Method(http.MethodHead, "/", HandleCollection(f))
			r.Method(http.MethodPost, "/", HandleActivity(f))

			r.Route("/{id}", func(r chi.Router) {
				r.Group(f.OAuthRoutes())
				r.Method(http.MethodGet, "/", HandleItem(f))
				r.Method(http.MethodHead, "/", HandleItem(f))
				if descend {
					r.Route("/{collection}", f.CollectionRoutes(false))
				}
			})
		})
	}
}

func (f FedBOX) Routes() func(chi.Router) {
	return func(r chi.Router) {
		r.Use(middleware.RealIP)
		r.Use(CleanRequestPath)

		r.Method(http.MethodGet, "/", HandleItem(f))
		r.Method(http.MethodHead, "/", HandleItem(f))
		// TODO(marius): we can separate here the FedBOX specific collections from the ActivityPub spec ones
		// using some regular expressions
		// Eg: "/{collection:(inbox|outbox|followed)}"
		// Eg: "/{collection:(activities|objects|actors|moderators|ignored|blocked|flagged)}"
		r.Route("/{collection}", f.CollectionRoutes(true))

		r.Group(f.OAuthRoutes())

		if f.conf.Env.IsDev() && !f.conf.Secure {
			r.Mount("/debug", middleware.Profiler())
		}

		r.Handle("/favicon.ico", errors.NotFound)
		r.NotFound(errors.NotFound.ServeHTTP)
		r.MethodNotAllowed(errors.HandleError(errors.MethodNotAllowedf("method not allowed")).ServeHTTP)
	}
}

func (f *FedBOX) OAuthRoutes() func(router chi.Router) {
	h := f.OAuth
	return func(r chi.Router) {
		r.Route("/oauth", func(r chi.Router) {
			// Authorization code endpoint
			r.Get("/authorize", h.Authorize)
			r.Post("/authorize", h.Authorize)
			// Access token endpoint
			r.Post("/token", h.Token)

			r.Group(func(r chi.Router) {
				r.Get("/login", h.ShowLogin)
				r.Post("/login", h.HandleLogin)
				r.Get("/pw", h.ShowChangePw)
				r.Post("/pw", h.HandleChangePw)
			})
		})
	}
}
