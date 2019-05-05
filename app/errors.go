package app

import (
	"github.com/go-ap/fedbox/internal/errors"
	"net/http"
)

// ErrorHandlerFn
type ErrorHandlerFn func(http.ResponseWriter, *http.Request) error

// ServeHTTP implements the http.Handler interface for the ItemHandlerFn type
func (h ErrorHandlerFn) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var dat []byte
	var status int

	if err := h(w, r); err != nil {
		status, dat = errors.Render(r, err)
	}

	w.Header().Set("Content-Type", "application/json+ld")
	w.WriteHeader(status)
	w.Write(dat)
}

func HandleError(e error) ErrorHandlerFn {
	return func(w http.ResponseWriter, r *http.Request) error {
		return e
	}
}
