package app

import (
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/fedbox/internal/context"
	"github.com/go-ap/fedbox/internal/errors"
	st "github.com/go-ap/fedbox/storage"
	"io/ioutil"
	"net/http"
)

// HandleClientRequest handles client to server (C2S) POST requests to an ActivityPub Actor's outbox
func HandleClientRequest(w http.ResponseWriter, r *http.Request) (as.IRI, int, error) {
	f := &st.Filters{}
	if err := f.FromRequest(r); err != nil {
		return as.IRI(""), http.StatusBadRequest, errors.BadRequestf("could not load filters from request")
	}

	var it as.Item
	var err error
	if body, err := ioutil.ReadAll(r.Body); err != nil || len(body) == 0 {
		return as.IRI(""), http.StatusInternalServerError, errors.NewNotValid(err, "unable to read request body")
	} else {
		if it, err = as.UnmarshalJSON(body); err != nil {
			return as.IRI(""), http.StatusInternalServerError, errors.NewNotValid(err, "unable to unmarshal JSON request")
		}
	}
	if repo, ok := context.ActivitySaver(r.Context()); ok == true {
		if it, err = repo.SaveActivity(it); err != nil {
			return as.IRI(""), http.StatusInternalServerError, errors.Annotatef(err, "Can't save %s %s", f.Collection, it.GetType())
		}
	}

	return it.GetLink(), http.StatusOK, nil
}
