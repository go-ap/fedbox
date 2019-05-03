package app

import (
	"errors"
	"github.com/go-chi/chi"
	"net/http"
)

type HandlerWithError func(w http.ResponseWriter, r *http.Request) error

type Handler HandlerWithError

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h(w, r); err != nil {
		// handle returned error here.
		w.WriteHeader(503)
		w.Write([]byte("bad"))
	}
}

func customHandler(w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query().Get("err")

	if q != "" {
		return errors.New(q)
	}

	w.Write([]byte("foo"))

	return nil
}

func Routes() func(chi.Router) {
	return func(r chi.Router) {
		r.Handle("/", Handler(customHandler))
	}
}
