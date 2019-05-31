package app

import (
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/errors"
	h "github.com/go-ap/handlers"
	"github.com/go-ap/storage"
	"io/ioutil"
	"net/http"
)

// HandleRequest handles POST requests to an ActivityPub Actor's inbox/outbox, based on the CollectionType
func HandleRequest(typ h.CollectionType, r *http.Request, repo storage.ActivitySaver) (as.IRI, int, error) {
	ff, _ := activitypub.FromRequest(r)
	f, _ := ff.(*activitypub.Filters)
	LoadToFilters(r, f)

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
	if err = validator.ValidateActivity(typ, it); err != nil {
		return as.IRI(""), http.StatusBadRequest, NewBadRequest(err, "%s activity failed validation", it.GetType())
	}

	if typ == h.Outbox {
		// C2S - get recipients and cleanup activity
		if actWRecipients, ok := it.(as.HasRecipients); ok {
			recipients := actWRecipients.Recipients()
			func (rec as.ItemCollection) {
				// TODO(marius): for C2S activities propagate them
			}(recipients)
		}
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
