package app

import (
	"github.com/go-ap/fedbox/internal/errors"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"net/http"
)

func ActivityRoutes(r chi.Router) {
	r.Group(func (r chi.Router) {
		r.With(middleware.GetHead)
		r.Method(http.MethodGet, "/", CollectionHandlerFn(HandleActivityCollection))
		r.Method(http.MethodGet, "/{id}", ItemHandlerFn(HandleActivityItem))
	})
}

func ObjectRoutes(r chi.Router) {
	r.Group(func (r chi.Router) {
		r.With(middleware.GetHead)
		r.Method(http.MethodGet, "/", CollectionHandlerFn(HandleObjectCollection))
		r.Route("/{id}", func (r chi.Router) {
			r.Method(http.MethodGet, "/", ItemHandlerFn(HandleObjectItem))
			r.Route("/{collection}",  ActivityRoutes)
		})
	})
}

func Routes() func(chi.Router) {
	return func(r chi.Router) {
		r.Route("/activities", ActivityRoutes)
		r.Route("/actors", ObjectRoutes)
		r.Route("/items",  ObjectRoutes)

		r.NotFound(HandleError(errors.MethodNotAllowedf("invalid url")).ServeHTTP)
		r.MethodNotAllowed(HandleError(errors.MethodNotAllowedf("method not allowed")).ServeHTTP)
	}
}
