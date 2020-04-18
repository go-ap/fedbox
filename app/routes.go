package app

import (
	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	h "github.com/go-ap/handlers"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/openshift/osin"
	"github.com/sirupsen/logrus"
	"net/http"
)

func CollectionRoutes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.With(middleware.GetHead)

		r.Method(http.MethodGet, "/", h.CollectionHandlerFn(HandleCollection))
		r.Method(http.MethodPost, "/", h.ActivityHandlerFn(HandleRequest))

		r.Route("/{id}", func(r chi.Router) {
			r.Method(http.MethodGet, "/", h.ItemHandlerFn(HandleItem))
			r.Route("/{collection}", func(r chi.Router) {
				r.Method(http.MethodGet, "/", h.CollectionHandlerFn(HandleCollection))
				r.Method(http.MethodPost, "/", h.ActivityHandlerFn(HandleRequest))
			})
		})
	})
}

var AnonymousAcct = account{
	username: "anonymous",
	actor:    &auth.AnonymousActor,
}

func (f *FedBOX) Routes(baseURL string, os *osin.Server, l logrus.FieldLogger) func(chi.Router) {
	return func(r chi.Router) {
		r.Use(middleware.RealIP)
		r.Use(middleware.GetHead)
		r.Use(ActorFromAuthHeader(os, f.Storage, l))

		r.Method(http.MethodGet, "/", h.ItemHandlerFn(HandleItem))
		r.Route("/{collection}", CollectionRoutes)

		h := oauthHandler{
			baseURL: baseURL,
			os:      os,
			loader:  f.Storage,
			logger:  l,
		}
		r.Route("/oauth", func(r chi.Router) {
			// Authorization code endpoint
			r.Get("/authorize", h.Authorize)
			r.Post("/authorize", h.Authorize)
			// Access token endpoint
			r.Post("/token", h.Token)

			r.Group(func(r chi.Router) {
				r.Get("/login", h.ShowLogin)
				r.Post("/login", h.HandleLogin)
				r.Get("/callback", h.HandleCallback)
				r.Get("/pw", h.ShowChangePw)
				r.Post("/pw", h.HandleChangePw)
			})
		})

		r.NotFound(errors.HandleError(errors.NotFoundf("invalid url")).ServeHTTP)
		r.MethodNotAllowed(errors.HandleError(errors.MethodNotAllowedf("method not allowed")).ServeHTTP)
	}
}
