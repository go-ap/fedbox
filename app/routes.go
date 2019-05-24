package app

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"net/http"
)

func CollectionRoutes(r chi.Router) {
	r.Group(func (r chi.Router) {
		r.With(middleware.GetHead)
		r.Method(http.MethodGet, "/", CollectionHandlerFn(HandleCollection))

		r.Route("/{id}", func (r chi.Router) {
			r.Method(http.MethodGet, "/", ItemHandlerFn(HandleItem))
			r.Method(http.MethodGet, "/{collection}", CollectionHandlerFn(HandleCollection))

			r.Group(func (r chi.Router) {
				val := genericValidator{}
				r.Use(Validator(&val))
				r.Method(http.MethodPost, "/{collection}", ActivityHandlerFn(HandleRequest))
			})
		})
	})
}

func Routes() func(chi.Router) {
	return func(r chi.Router) {
		r.Use(middleware.GetHead)
		r.Use(ActorFromAuthHeader)

		r.Route("/{collection}", CollectionRoutes)

		r.NotFound(HandleError(NotFoundf("invalid url")).ServeHTTP)
		r.MethodNotAllowed(HandleError(MethodNotAllowedf("method not allowed")).ServeHTTP)
	}
}
