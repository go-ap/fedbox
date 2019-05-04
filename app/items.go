package app

import (
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/fedbox/internal/errors"
	j "github.com/go-ap/jsonld"
	"github.com/go-chi/chi"
	"net/http"
)

func renderItem(i as.Item) ([]byte, error) {
	return j.Marshal(i)
}

// ItemHandlerFn is the type that we're using to represent handlers that return ActivityStreams
// objects. It needs to implement the http.Handler interface
type ItemHandlerFn func(http.ResponseWriter, *http.Request) (as.Item, error)

// ValidMethod validates if the current handler can process the current request
func (i ItemHandlerFn) ValidMethod( r *http.Request) bool {
	return r.Method != http.MethodGet && r.Method != http.MethodHead
}

// ServeHTTP implements the http.Handler interface for the ItemHandlerFn type
func (i ItemHandlerFn) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var dat []byte
	var status int

	if i.ValidMethod(r) {
		status = http.StatusNotAcceptable
		dat, _ = errors.Render(errors.MethodNotAllowedf("invalid HTTP method"))
	}

	if it, err := i(w, r); err != nil {
		// HandleError
		status = http.StatusInternalServerError
		dat, _ = errors.Render(err)
	} else {
		if dat, err = renderItem(it); err != nil {
			status = http.StatusInternalServerError
			dat, _ = errors.Render(err)
		} else {
			status = http.StatusOK
		}
	}

	w.WriteHeader(status)
	if r.Method == http.MethodGet {
		w.Write(dat)
	}
}

// HandleActivityItem serves content from the outbox, inbox, likes, shares and replies end-points
// that returns a single ActivityPub activity
func HandleActivityItem(w http.ResponseWriter, r *http.Request) (as.Item, error) {
	// TODO(marius): move typer instantiation outside the handler, so we can pass it from outside
	typer := DefaultCollectionTyper{}
	collection :=  typer.Type(r)

	id := chi.URLParam(r, "id")
	return nil, errors.NotFoundf("%s %s", collection, id)
}

// HandleObjectItem serves content from the following, followers, liked, and likes end-points
// that returns a single ActivityPub object
func HandleObjectItem(w http.ResponseWriter, r *http.Request) (as.Item, error) {
	// TODO(marius): move typer instantiation outside the handler, so we can pass it from outside
	typer := DefaultCollectionTyper{}
	collection :=  typer.Type(r)

	id := chi.URLParam(r, "id")
	return nil, errors.NotFoundf("%s %s", collection, id)
}
