package app

import (
	h "github.com/go-ap/activitypub/handler"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/fedbox/internal/errors"
	"net/http"
)

// HandleClientRequest handles client to server (C2S) POST requests to an ActivityPub Actor's outbox
func HandleClientRequest(w http.ResponseWriter, r *http.Request) (as.IRI, int, error) {
	collection := h.Typer.Type(r)
	return as.IRI(""), http.StatusNotImplemented, errors.NotImplementedf("%s", collection)
}
