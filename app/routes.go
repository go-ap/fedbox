package app

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"net/http"
)

type HandlerWithError func(w http.ResponseWriter, r *http.Request) error

type Handler HandlerWithError

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
		r.Method(http.MethodGet, "/{id}", ItemHandlerFn(HandleObjectItem))
	})
}

func Routes() func(chi.Router) {
	return func(r chi.Router) {
		r.Route("/activities", ActivityRoutes)
		r.Route("/actors", ObjectRoutes)
		r.Route("/items",  ObjectRoutes)
	}
}
