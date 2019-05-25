package app

import (
	"fmt"
	"github.com/go-ap/activitypub/handler"
	as "github.com/go-ap/activitystreams"
	j "github.com/go-ap/jsonld"
	"net/http"
)

// ItemHandlerFn is the type that we're using to represent handlers that return ActivityStreams
// objects. It needs to implement the http.Handler interface
type ItemHandlerFn func(http.ResponseWriter, *http.Request) (as.Item, error)

// ValidMethod validates if the current handler can process the current request
func (i ItemHandlerFn) ValidMethod(r *http.Request) bool {
	return r.Method == http.MethodGet || r.Method == http.MethodHead
}

// ValidateRequest validates if the current handler can process the current request
func (i ItemHandlerFn) ValidateRequest(r *http.Request) (int, error) {
	if !i.ValidMethod(r) {
		return http.StatusMethodNotAllowed, fmt.Errorf("invalid HTTP method %s", r.Method)
	}
	return http.StatusOK, nil
}

// ServeHTTP implements the http.Handler interface for the ItemHandlerFn type
func (i ItemHandlerFn) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var dat []byte
	var status = http.StatusOK
	var err error

	if status, err = i.ValidateRequest(r); err != nil {
		_, dat = RenderErrors(r, err)
	} else {
		if it, err := i(w, r); err != nil {
			status, dat = RenderErrors(r, err)
		} else {
			if dat, err = j.WithContext(j.IRI(as.ActivityBaseURI)).Marshal(it); err != nil {
				status, dat = RenderErrors(r, err)
			}
		}
	}

	w.Header().Set("Content-Type", "application/ld+json")
	w.WriteHeader(status)
	if r.Method == http.MethodGet {
		w.Write(dat)
	}
}
// ActivityHandlerFn is the type that we're using to represent handlers that process requests containing
// an ActivityStreams Activity. It needs to implement the http.Handler interface.
type ActivityHandlerFn func(handler.CollectionType, *http.Request) (as.IRI, int, error)

// ValidateRequest validates if the current handler can process the current request
func (a ActivityHandlerFn) ValidateRequest(r *http.Request) (int, error) {
	return http.StatusOK, nil
}
// ServeHTTP implements the http.Handler interface for the CollectionHandlerFn type
func (a ActivityHandlerFn) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var dat []byte
	var iri as.IRI
	var err error
	var status = http.StatusOK

	if status, err = a.ValidateRequest(r); err != nil {
		_, dat = RenderErrors(r, err)
	}

	if iri, status, err = a(handler.Typer.Type(r), r); err != nil {
		_, dat = RenderErrors(r, err)
	} else {
		dat, _ = j.Marshal("OK")
	}

	w.Header().Set("Content-Type", "application/ld+json")
	if len(iri) > 0 {
		w.Header().Set("Location", iri.String())
	}
	w.WriteHeader(status)
	w.Write(dat)
}

// CollectionHandlerFn is the type that we're using to represent handlers that will return ActivityStreams
// Collection or OrderedCollection objects. It needs to implement the http.Handler interface.
type CollectionHandlerFn func(*http.Request) (as.CollectionInterface, error)

// ValidMethod validates if the current handler can process the current request
func (c CollectionHandlerFn) ValidMethod(r *http.Request) bool {
	return r.Method == http.MethodGet || r.Method == http.MethodHead
}

// ValidateRequest validates if the current handler can process the current request
func (c CollectionHandlerFn) ValidateRequest(r *http.Request) (int, error) {
	if !c.ValidMethod(r) {
		return http.StatusMethodNotAllowed, fmt.Errorf("invalid HTTP method %s", r.Method)
	}
	return http.StatusOK, nil
}

// ServeHTTP implements the http.Handler interface for the CollectionHandlerFn type
func (c CollectionHandlerFn) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var dat []byte

	var status = http.StatusOK
	var err error

	if status, err = c.ValidateRequest(r); err != nil {
		_, dat = RenderErrors(r, err)
	} else {
		if col, err := c(r); err != nil {
			status, dat = RenderErrors(r, err)
		} else {
			if dat, err = j.WithContext(j.IRI(as.ActivityBaseURI)).Marshal(col); err != nil {
				status, dat = RenderErrors(r, err)
			}
		}
	}

	w.Header().Set("Content-Type", "application/ld+json")
	w.WriteHeader(status)
	if r.Method == http.MethodGet {
		w.Write(dat)
	}
}
