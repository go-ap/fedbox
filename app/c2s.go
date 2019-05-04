package app

import (
	as "github.com/go-ap/activitystreams"
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
	var ob, act as.Item
	if repo, ok := ContextActivitySaver(r.Context()); ok == true {
		if it, err = repo.SaveActivity(it); err != nil {
			return as.IRI(""), http.StatusInternalServerError, errors.Annotatef(err, "Can't save %s %s", f.Collection, it.GetType())
		}
		if a := as.ToActivity(it); ok {
			ob = a.Object
			act = a.Actor
		}
	}
	if repo, ok := ContextActorSaver(r.Context()); ok == true && act != nil {
		if it, err = repo.SaveActor(act); err != nil {
			return as.IRI(""), http.StatusInternalServerError, errors.Annotatef(err, "Can't save %s %s", f.Collection, act.GetType())
		}
	}
	if repo, ok := ContextObjectSaver(r.Context()); ok == true && ob != nil {
		if it, err = repo.SaveObject(ob); err != nil {
			return as.IRI(""), http.StatusInternalServerError, errors.Annotatef(err, "Can't save %s %s", f.Collection, ob.GetType())
		}
	}

	return it.GetLink(), http.StatusOK, nil
}
