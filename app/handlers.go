package app

import (
	"fmt"
	ap "github.com/go-ap/activitypub"
	"github.com/go-ap/activitypub/client"
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

func validContentType(c string) bool {
	if c == client.ContentTypeActivityJson || c == client.ContentTypeJsonLD {
		return true
	}

	return false
}

func ValidateRequest(r *http.Request) (bool, error) {
	contType := r.Header.Get("Content-Type")
	if validContentType(contType) {
		return true, nil
	}

	return false, errors.Newf("Invalid request")
}

// HandleRequest handles POST requests to an ActivityPub To's inbox/outbox, based on the CollectionType
func HandleRequest(typ h.CollectionType, r *http.Request, repo storage.Repository) (as.Item, int, error) {
	var err error
	var it as.Item

	f, err := activitypub.FromRequest(r)
	if err != nil {
		return it, 0, errors.NewNotValid(err, "unable to load filters from request")
	}
	LoadCollectionFilters(r, f)

	if ok, err := ValidateRequest(r); !ok {
		return it, http.StatusInternalServerError, errors.NewNotValid(err, "unrecognized ActivityPub request")
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		return it, http.StatusInternalServerError, errors.NewNotValid(err, "unable to read request body")
	}
	if it, err = as.UnmarshalJSON(body); err != nil {
		return it, http.StatusInternalServerError, errors.NewNotValid(err, "unable to unmarshal JSON request")
	}
	validator, ok := validation.FromContext(r.Context())
	if ok == false {
		return it, http.StatusInternalServerError, errors.Annotatef(err, "unable to load activity validator")
	}
	validator.SetActor(f.Authenticated)

	var validateFn func(as.Item, as.IRI) error
	switch typ {
	case h.Outbox:
		validateFn = validator.ValidateClientActivity
	case h.Inbox:
		validateFn = validator.ValidateServerActivity
	default:
		return it, http.StatusNotAcceptable, errors.NewMethodNotAllowed(err, "Collection %s does not receive Activity requests", typ)
	}
	if err = validateFn(it, f.IRI); err != nil {
		return it, 0, err
	}
	ap.OnActivity(it, func(a *as.Activity) error {
		// TODO(marius): this should be handled in the processing package
		if a.AttributedTo == nil {
			a.AttributedTo = f.Authenticated
		}
		return nil
	})

	act, err := as.ToActivity(it)
	if err != nil {
		return it, http.StatusInternalServerError, errors.Annotatef(err, "Invalid activity %s", it.GetType())
	}
	if it, err = processing.ProcessActivity(repo, act, typ); err != nil {
		return it, http.StatusInternalServerError, errors.Annotatef(err, "Can't save activity %s to %s", it.GetType(), f.Collection)
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

	status := http.StatusOK
	if it.GetType() == as.DeleteType {
		status = http.StatusGone
	}
	if it.GetType() == as.CreateType {
		status = http.StatusCreated
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
		where = fmt.Sprintf(" in %s", collection)
	}
	if err != nil {
		return nil, errors.NotFoundf("%snot found", what)
	}
	LoadItemFilters(r, f)

	iri := reqURL(r)
	if len(f.IRI) == 0 {
		f.IRI = as.IRI(iri)
	}
	what = fmt.Sprintf("%s ", path.Base(iri))
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
			fallthrough
		default:
			items, _, err = repo.LoadObjects(f)
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
