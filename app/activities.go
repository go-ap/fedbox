package app

import (
	h "github.com/go-ap/activitypub/handler"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/fedbox/internal/context"
	"github.com/go-ap/fedbox/internal/errors"
	st "github.com/go-ap/fedbox/storage"
	"io/ioutil"
	"net/http"
)

type ActivityPubActivityHandler interface {
	ActivityPubActivityC2SHandler
	ActivityPubActivityS2SHandler
}

type ActivityPubActivityC2SHandler interface {
	HandleClientRequest(http.ResponseWriter, *http.Request) (as.IRI, int, error)
}
type ActivityPubActivityS2SHandler interface {
	HandleServerRequest(http.ResponseWriter, *http.Request) (as.IRI, int, error)
}

type DumbHandler struct {}

// HandleClientRequest handles client to server (C2S) POST requests to an ActivityPub Actor's outbox
func HandleClientRequest(w http.ResponseWriter, r *http.Request) (as.IRI, int, error) {
	f := &st.Filters{}
	if err := f.FromRequest(r); err != nil {
		return as.IRI(""), http.StatusBadRequest, BadRequestf("could not load filters from request")
	}

	var it as.Item
	var err error
	if body, err := ioutil.ReadAll(r.Body); err != nil || len(body) == 0 {
		return as.IRI(""), http.StatusInternalServerError, NewNotValid(err, "unable to read request body")
	} else {
		if it, err = as.UnmarshalJSON(body); err != nil {
			return as.IRI(""), http.StatusInternalServerError, NewNotValid(err, "unable to unmarshal JSON request")
		}
	}
	if repo, ok := context.ActivitySaver(r.Context()); ok == true {
		if it, err = repo.SaveActivity(it); err != nil {
			return as.IRI(""), http.StatusInternalServerError, errors.Annotatef(err, "Can't save %s %s", f.Collection, it.GetType())
		}
	}
	status := http.StatusOK
	if it.GetType() == as.CreateType {
		status = http.StatusCreated
	}
	if it.GetType() == as.DeleteType {
		status = http.StatusGone
	}

	return it.GetLink(), status, nil
}

// HandleServerRequest handles server to server (S2S) POST requests to an ActivityPub Actor's inbox
func HandleServerRequest(w http.ResponseWriter, r *http.Request) (as.IRI, int, error) {
	f := &st.Filters{}
	f.FromRequest(r)

	if f.Collection != h.Inbox {
		return as.IRI(""), http.StatusNotFound, NotFoundf("%s", f.Collection)
	}

	var it as.Item
	var err error
	if body, err := ioutil.ReadAll(r.Body); err != nil || len(body) == 0 {
		return as.IRI(""), http.StatusInternalServerError, NewNotValid(err, "unable to read request body")
	} else {
		if it, err = as.UnmarshalJSON(body); err != nil {
			return as.IRI(""), http.StatusInternalServerError, NewNotValid(err, "unable to unmarshal JSON request")
		}
	}
	validator, ok := ActivityValidatorCtxt(r.Context())
	if ok == false {
		return as.IRI(""), http.StatusInternalServerError, errors.Annotatef(err, "Unable to load activity validator")
	}
	if err = validator.ValidateActivity(it); err != nil {
		return as.IRI(""), http.StatusBadRequest, NewBadRequest(err, "%s activity failed validation", it.GetType())
	}
	repo, ok := context.ActivitySaver(r.Context())
	if  ok == false {
		return as.IRI(""), http.StatusInternalServerError, errors.Annotatef(err, "Unable to load activity saver")
	}
	if it, err = repo.SaveActivity(it); err != nil {
		return as.IRI(""), http.StatusInternalServerError, errors.Annotatef(err, "Can't save activity %s to %s", it.GetType(), f.Collection)
	}
	status := http.StatusOK
	if it.GetType() == as.CreateType {
		status = http.StatusCreated
	}
	if it.GetType() == as.DeleteType {
		status = http.StatusGone
	}

	return it.GetLink(), status, nil
}
