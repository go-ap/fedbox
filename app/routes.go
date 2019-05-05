package app

import (
	h "github.com/go-ap/activitypub/handler"
	"github.com/go-ap/fedbox/internal/errors"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"net/http"
)

func ActivityRoutes(r chi.Router) {
	r.Group(func (r chi.Router) {
		r.With(middleware.GetHead)
		r.Method(http.MethodGet, "/", h.CollectionHandlerFn(HandleActivityCollection))
		r.Method(http.MethodGet, "/{id}", h.ItemHandlerFn(HandleActivityItem))
	})
}

func ObjectRoutes(r chi.Router) {
	r.Group(func (r chi.Router) {
		r.With(middleware.GetHead)
		r.Method(http.MethodGet, "/", h.CollectionHandlerFn(HandleObjectCollection))
		r.Route("/{id}", func (r chi.Router) {
			r.Method(http.MethodGet, "/", h.ItemHandlerFn(HandleObjectItem))
			r.Route("/replies",  ActivityRoutes)
		})
	})
}

func ActorRoutes(r chi.Router) {
	r.Group(func (r chi.Router) {
		r.Use(middleware.GetHead)
		r.Method(http.MethodGet, "/", CollectionHandlerFn(HandleObjectCollection))
		r.Route("/{id}", func (r chi.Router) {
			r.Method(http.MethodGet, "/", ItemHandlerFn(HandleObjectItem))
			r.Route("/{collection}", ActivityRoutes)

			r.Group(func (r chi.Router) {
				r.Use(ActorFromAuthHeader)
				r.Method(http.MethodPost, "/inbox", ActivityHandlerFn(HandleServerRequest))
				r.Method(http.MethodPost, "/outbox", ActivityHandlerFn(HandleClientRequest))
			})
		})
	})
}

func Routes() func(chi.Router) {
	return func(r chi.Router) {
		r.Route("/activities", ActivityRoutes)
		r.Route("/actors", ActorRoutes)
		r.Route("/items",  ObjectRoutes)

		r.NotFound(HandleError(errors.MethodNotAllowedf("invalid url")).ServeHTTP)
		r.MethodNotAllowed(HandleError(errors.MethodNotAllowedf("method not allowed")).ServeHTTP)
	}
}
