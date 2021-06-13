package app

import (
	"net/http"

	"github.com/go-ap/errors"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

func (f FedBOX) CollectionRoutes(descend bool) func(chi.Router) {
	return func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.With(f.ActorFromAuthHeader).Method(http.MethodGet, "/", HandleCollection(f))
			r.With(f.ActorFromAuthHeader).Method(http.MethodHead, "/", HandleCollection(f))
			r.With(f.ActorFromAuthHeader).Method(http.MethodPost, "/", HandleRequest(f))

			r.Route("/{id}", func(r chi.Router) {
				r.Group(f.OAuthRoutes())
				r.With(f.ActorFromAuthHeader).Method(http.MethodGet, "/", HandleItem(f))
				r.With(f.ActorFromAuthHeader).Method(http.MethodHead, "/", HandleItem(f))
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

		r.With(f.ActorFromAuthHeader).Method(http.MethodGet, "/", HandleItem(f))
		r.With(f.ActorFromAuthHeader).Method(http.MethodHead, "/", HandleItem(f))
		r.Route("/{collection}", f.CollectionRoutes(true))

		r.Group(f.OAuthRoutes())

		notFound := errors.HandleError(errors.NotFoundf("invalid url"))
		r.Handle("/favicon.ico", notFound)
		r.NotFound(notFound.ServeHTTP)
		r.MethodNotAllowed(errors.HandleError(errors.MethodNotAllowedf("method not allowed")).ServeHTTP)
	}
}

func (f *FedBOX) OAuthRoutes() func(router chi.Router) {
	h := f.OAuth
	return func (r chi.Router) {
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
