package app

import (
	"github.com/go-chi/chi"
	"net/http"
)

type HandlerWithError func(w http.ResponseWriter, r *http.Request) error

type Handler HandlerWithError

func Routes() func(chi.Router) {
	return func(r chi.Router) {
		r.Handle("/activities", CollectionHandlerFn(HandleActivityCollection))
		r.Handle("/actors", CollectionHandlerFn(HandleObjectCollection))
		r.Handle("/items", CollectionHandlerFn(HandleObjectCollection))
	}
}
