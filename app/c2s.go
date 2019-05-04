package app

import (
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/fedbox/internal/errors"
	"net/http"
)

// ActivityHandlerFn is the type that we're using to represent handlers that process requests containing
// an ActivityStreams Activity. It needs to implement the http.Handler interface
type ActivityHandlerFn func(http.ResponseWriter, *http.Request) (as.IRI, int, error)

// ValidMethod validates if the current handler can process the current request
func (a ActivityHandlerFn) ValidMethod( r *http.Request) bool {
	return r.Method != http.MethodPost
}

// ServeHTTP implements the http.Handler interface for the ActivityHandlerFn type
func (a ActivityHandlerFn) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var dat []byte
	var status int
	var iri as.IRI
	var err error

	if !a.ValidMethod(r) {
		status = http.StatusNotAcceptable
		dat, _ = errors.Render(errors.MethodNotAllowedf("invalid HTTP method"))
	}

	if iri, status, err = a(w, r); err != nil {
		// HandleError
		status = http.StatusInternalServerError
		dat, _ = errors.Render(err)
	} else {
		dat = []byte("OK")
	}

	w.WriteHeader(status)
	if len(iri) > 0 {
		w.Header().Set("Location", iri.String())
	}
	w.Write(dat)
}

// HandleClientRequest handles client to server (C2S) POST requests to an ActivityPub Actor's outbox
func HandleClientRequest(w http.ResponseWriter, r *http.Request) (as.IRI, int, error) {
	collection :=  Typer.Type(r)
	return as.IRI(""), http.StatusNotImplemented, errors.NotImplementedf("%s", collection)
}
