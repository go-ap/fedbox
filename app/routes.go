package app

import (
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
			r.Route("/replies",  ActivityRoutes)
		})
	})
}

func ActorRoutes(r chi.Router) {
	r.Group(func (r chi.Router) {

		r.Method(http.MethodGet, "/", CollectionHandlerFn(HandleObjectCollection))
		r.Route("/{id}", func (r chi.Router) {
			r.Method(http.MethodGet, "/", ItemHandlerFn(HandleObjectItem))
			r.Route("/{collection}", ActivityRoutes)

			val := genericValidator{}
			r.Group(func (r chi.Router) {
				r.Use(Validator(&val))
				r.Method(http.MethodPost, "/inbox", ActivityHandlerFn(HandleServerRequest))
				r.Method(http.MethodPost, "/outbox", ActivityHandlerFn(HandleClientRequest))
			})
		})
	})
}

func Routes() func(chi.Router) {
	return func(r chi.Router) {
		r.Use(middleware.GetHead)
		r.Use(ActorFromAuthHeader)

		r.Route("/activities", ActivityRoutes)
		r.Route("/actors", ActorRoutes)
		r.Route("/objects",  ObjectRoutes)

		val := genericValidator{}
		r.With(Validator(&val)).Method(http.MethodPost, "/inbox", ActivityHandlerFn(HandleServerRequest))

		r.NotFound(HandleError(NotFoundf("invalid url")).ServeHTTP)
		r.MethodNotAllowed(HandleError(MethodNotAllowedf("method not allowed")).ServeHTTP)
	}
}
