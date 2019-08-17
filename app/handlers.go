package app

import (
	"fmt"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/validation"
	h "github.com/go-ap/handlers"
	"github.com/go-ap/processing"
	"github.com/go-ap/storage"
	"io/ioutil"
	"net/http"
	"path"
)

func reqURL(r *http.Request) string {
	scheme := "http"
	if Config.Secure || r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL.Path)
}

// HandleCollection serves content from the generic collection end-points
// that return ActivityPub objects or activities
func HandleCollection(typ h.CollectionType, r *http.Request, repo storage.CollectionLoader) (as.CollectionInterface, error) {
	var items as.CollectionInterface
	var err error

	f, err := activitypub.FromRequest(r)
	if err != nil {
		return nil, errors.NewNotValid(err, "Unable to load filters from request")
	}
	LoadCollectionFilters(r, f)

	if !activitypub.ValidActivityCollection(string(typ)) {
		return nil, errors.NotFoundf("collection '%s' not found", f.Collection)
	}

	items, err = repo.LoadCollection(f)
	if err != nil {
		return nil, err
	}
	return activitypub.PaginateCollection(items, f)
}

// HandleRequest handles POST requests to an ActivityPub To's inbox/outbox, based on the CollectionType
func HandleRequest(typ h.CollectionType, r *http.Request, repo storage.Repository) (as.Item, int, error) {
	var err error
	var it as.Item

	f, err := activitypub.FromRequest(r)
	if err != nil {
		return it, 0, errors.NewNotValid(err, "Unable to load filters from request")
	}
	LoadCollectionFilters(r, f)

	if body, err := ioutil.ReadAll(r.Body); err != nil || len(body) == 0 {
		return it, http.StatusInternalServerError, errors.NewNotValid(err, "unable to read request body")
	} else {
		if it, err = as.UnmarshalJSON(body); err != nil {
			return it, http.StatusInternalServerError, errors.NewNotValid(err, "unable to unmarshal JSON request")
		}
	}
	validator, ok := validation.FromContext(r.Context())
	if ok == false {
		return it, http.StatusInternalServerError, errors.Annotatef(err, "Unable to load activity validator")
	}
	validator.SetActor(f.Authenticated)
	var validateFn func(as.Item, as.IRI) error
	switch typ {
	case h.Outbox :
		validateFn = validator.ValidateClientActivity
	case h.Inbox:
		validateFn = validator.ValidateServerActivity
	default:
		return it, http.StatusNotAcceptable, errors.NewMethodNotAllowed(err, "Collection %s does not receive Activity requests", typ)
	}
	if err = validateFn(it, f.IRI); err != nil {
		return it, 0, err
	}

	if typ == h.Outbox {
		// C2S - get recipients and cleanup activity
		if actWRecipients, ok := it.(as.HasRecipients); ok {
			recipients := actWRecipients.Recipients()
			func(rec as.ItemCollection) {
				// TODO(marius): for C2S activities propagate them
			}(recipients)
		}
	}

	act, err := as.ToActivity(it)
	if err != nil {
		return it, http.StatusInternalServerError, errors.Annotatef(err, "Invalid activity %s", it.GetType())
	}
	if it, err = processing.ProcessActivity(repo, act); err != nil {
		return it, http.StatusInternalServerError, errors.Annotatef(err, "Can't save activity %s to %s", it.GetType(), f.Collection)
	}

	status := http.StatusCreated
	if it.GetType() == as.DeleteType {
		status = http.StatusGone
	}
	if it.GetType() == as.UpdateType {
		status = http.StatusOK
	}

	return it, status, nil
}

// HandleItem serves content from the following, followers, liked, and likes end-points
// that returns a single ActivityPub object
func HandleItem(r *http.Request, repo storage.ObjectLoader) (as.Item, error) {
	collection := h.Typer.Type(r)

	var items as.ItemCollection
	var err error
	f, err := activitypub.FromRequest(r)

	where := ""
	what := ""
	if len(collection) > 0 {
		what = fmt.Sprintf("%s ", collection)
		where = fmt.Sprintf(" in %s", what)
	}
	if err != nil {
		return nil, errors.NotFoundf("%snot found", what)
	}
	LoadItemFilters(r, f)

	iri := reqURL(r)
	if len(f.IRI) == 0 {
		f.IRI = as.IRI(iri)
	}
	f.MaxItems = 1

	if activitypub.ValidActivityCollection(string(f.Collection)) {
		switch f.Collection {
		case activitypub.ActivitiesType:
			if actLoader, ok := repo.(storage.ActivityLoader); ok {
				items, _, err = actLoader.LoadActivities(f)
			}
		case activitypub.ActorsType:
			if actLoader, ok := repo.(storage.ActorLoader); ok {
				items, _, err = actLoader.LoadActors(f)
			}
		case activitypub.ObjectsType:
			items, _, err = repo.LoadObjects(f)
		default:

		}
	} else if f.Collection == "" {
		if actLoader, ok := repo.(storage.ActorLoader); ok {
			items, _, err = actLoader.LoadActors(f)
		}
	}

	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.NotFoundf("%snot found%s", what, where)
	}
	if len(items) > 1 {
		what = fmt.Sprintf("%s", path.Base(iri))
		return nil, errors.Errorf("Too many %s found%s", what, where)
	}
	it, err := loadItem(items, f, reqURL(r))
	if err != nil {
		return nil, errors.NotFoundf("%snot found", what)
	}
	return it, nil
}

func loadItem(items as.ItemCollection, f activitypub.Paginator, baseURL string) (as.Item, error) {
	return items.First(), nil
}
