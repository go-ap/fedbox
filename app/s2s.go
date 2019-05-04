package app

import (
	h "github.com/go-ap/activitypub/handler"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/fedbox/internal/errors"
	st "github.com/go-ap/fedbox/storage"
	"net/http"
)

// HandleServerRequest handles server to server (S2S) POST requests to an ActivityPub Actor's inbox
func HandleServerRequest(w http.ResponseWriter, r *http.Request) (as.IRI, int, error) {
	f := st.Filters{}
	if err := f.FromRequest(r); err != nil {
		return as.IRI(""), http.StatusBadRequest, errors.BadRequestf("could not load filters from request")
	}

	if f.Collection != h.Inbox {
		return as.IRI(""), http.StatusNotFound, errors.NotFoundf("%s", f.Collection)
	}

	return as.IRI(""), http.StatusNotImplemented, errors.NotImplementedf("%s", f.Collection)
}
