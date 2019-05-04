package app

import (
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/fedbox/internal/errors"
	st "github.com/go-ap/fedbox/storage"
	"net/http"
)

// HandleClientRequest handles client to server (C2S) POST requests to an ActivityPub Actor's outbox
func HandleClientRequest(w http.ResponseWriter, r *http.Request) (as.IRI, int, error) {
	f := &st.Filters{}
	if err := f.FromRequest(r); err != nil {
		return as.IRI(""), http.StatusBadRequest, errors.BadRequestf("could not load filters from request")
	}

	return as.IRI(""), http.StatusNotImplemented, errors.NotImplementedf("%s", f.Collection)
}
