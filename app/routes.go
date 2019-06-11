package app

import (
	"github.com/go-ap/errors"
	h "github.com/go-ap/handlers"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"net/http"
)

func CollectionRoutes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.With(middleware.GetHead)
		val := genericValidator{}

		r.Method(http.MethodGet, "/", h.CollectionHandlerFn(HandleCollection))
		r.With(Validator(&val)).Method(http.MethodPost, "/", h.ActivityHandlerFn(HandleRequest))

		r.Route("/{id}", func(r chi.Router) {
			r.Method(http.MethodGet, "/", h.ItemHandlerFn(HandleItem))
			r.Route("/{collection}", func(r chi.Router) {
				r.Method(http.MethodGet, "/", h.CollectionHandlerFn(HandleCollection))
				r.With(Validator(&val)).Method(http.MethodPost, "/", h.ActivityHandlerFn(HandleRequest))
			})
		})
	})
}

func Routes() func(chi.Router) {
	return func(r chi.Router) {
		r.Use(middleware.GetHead)
		r.Use(ActorFromAuthHeader)

		r.Route("/{collection}", CollectionRoutes)

		r.NotFound(errors.HandleError(errors.NotFoundf("invalid url")).ServeHTTP)
		r.MethodNotAllowed(errors.HandleError(errors.MethodNotAllowedf("method not allowed")).ServeHTTP)
	}
}
