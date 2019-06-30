package app

import (
	"fmt"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/activitypub"
	h "github.com/go-ap/handlers"
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
	LoadItemFilters(r, &f)

	if !activitypub.ValidActivityCollection(string(typ)) {
		return nil, errors.NotFoundf("collection '%s' not found", f.Collection)
	}

	items, err = repo.LoadCollection(f)
	if err != nil {
		topLevelCollection := func(f activitypub.Filters) bool {
			return f.Author == nil && f.Parent == nil && f.To == nil
		}(f)
		if topLevelCollection && (f.Collection == h.Inbox || f.Collection == h.Outbox) {
			return nil, errors.MethodNotAllowedf("method not allowed")
		}
		return nil, err
	}
	return activitypub.PaginateCollection(items, &f)
}

// HandleRequest handles POST requests to an ActivityPub To's inbox/outbox, based on the CollectionType
func HandleRequest(typ h.CollectionType, r *http.Request, repo storage.Repository) (as.Item, int, error) {
	var err error
	var it as.Item

	f, err := activitypub.FromRequest(r)
	if err != nil {
		return it, 0, errors.NewNotValid(err, "Unable to load filters from request")
	}
	LoadItemFilters(r, &f)

	if body, err := ioutil.ReadAll(r.Body); err != nil || len(body) == 0 {
		return it, http.StatusInternalServerError, errors.NewNotValid(err, "unable to read request body")
	} else {
		if it, err = as.UnmarshalJSON(body); err != nil {
			return it, http.StatusInternalServerError, errors.NewNotValid(err, "unable to unmarshal JSON request")
		}
	}
	validator, ok := ActivityValidatorCtxt(r.Context())
	if ok == false {
		return it, http.StatusInternalServerError, errors.Annotatef(err, "Unable to load activity validator")
	}
	if err = validator.ValidateActivity(typ, it); err != nil {
		return it, http.StatusBadRequest, errors.NewBadRequest(err, "%s activity failed validation", it.GetType())
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

	if it, err = activitypub.ProcessActivity(repo, it); err != nil {
		return it, http.StatusInternalServerError, errors.Annotatef(err, "Can't save activity %s to %s", it.GetType(), f.Collection)
	}

	status := http.StatusCreated
	if it.GetType() == as.DeleteType {
		status = http.StatusGone
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
	if err != nil {
		return nil, errors.NotFoundf("Not found %s", collection)
	}
	LoadItemFilters(r, &f)

	iri := reqURL(r)
	if len(f.ItemKey) == 0 {
		f.ItemKey = []activitypub.Hash{
			activitypub.Hash(iri),
		}
	}
	f.MaxItems = 1

	if !activitypub.ValidActivityCollection(string(f.Collection)) {
		return nil, errors.NotFoundf("collection '%s' not found", f.Collection)
	}
	if h.ValidObjectCollection(string(f.Collection)) {
		if obLoader, ok := repo.(storage.ObjectLoader); ok {
			items, _, err = obLoader.LoadObjects(f)
		}
	} else if h.ValidActivityCollection(string(f.Collection)) {
		if actLoader, ok := repo.(storage.ActivityLoader); ok {
			items, _, err = actLoader.LoadActivities(f)
		}
	} else {
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
			if obLoader, ok := repo.(storage.ObjectLoader); ok {
				items, _, err = obLoader.LoadObjects(f)
			}
		default:
			return nil, errors.Newf("invalid collection %s", f.Collection)
		}
	}
	if err != nil {
		return nil, err
	}
	if len(items) == 1 {
		it, err := loadItem(items, f, reqURL(r))
		if err != nil {
			return nil, errors.NotFoundf("Not found %s", collection)
		}
		return it, nil
	}

	id := path.Base(iri)
	return nil, errors.NotFoundf("Not found %s in %s", id, collection)
}

func loadItem(items as.ItemCollection, f activitypub.Paginator, baseURL string) (as.Item, error) {
	return items[0], nil
}
