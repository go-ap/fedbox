package app

import (
	"github.com/go-chi/chi"
	"net/http"
)

type HandlerWithError func(w http.ResponseWriter, r *http.Request) error

type Handler HandlerWithError

func Routes() func(chi.Router) {
	return func(r chi.Router) {
		r.Route("/activities", func(r chi.Router) {
			r.Handle("/", CollectionHandlerFn(HandleActivityCollection))
			r.Handle("/{id}", ItemHandlerFn(HandleActivityItem))
		})
		r.Route("/actors", func(r chi.Router) {
			r.Handle("/", CollectionHandlerFn(HandleObjectCollection))
			r.Handle("/{id}", ItemHandlerFn(HandleObjectItem))
		})
		r.Route("/items",  func(r chi.Router) {
			r.Handle("/", CollectionHandlerFn(HandleObjectCollection))
			r.Handle("/{id}", ItemHandlerFn(HandleObjectItem))
		})
	}
}
