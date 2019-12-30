package app

import (
	"fmt"
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/client"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/activitypub"
	st "github.com/go-ap/fedbox/storage"
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
func HandleCollection(typ h.CollectionType, r *http.Request, repo storage.CollectionLoader) (pub.CollectionInterface, error) {
	var items pub.CollectionInterface
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
func HandleRequest(typ h.CollectionType, r *http.Request, repo storage.Repository) (pub.Item, int, error) {
	var err error
	var it pub.Item

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
	if it, err = pub.UnmarshalJSON(body); err != nil {
		return it, http.StatusInternalServerError, errors.NewNotValid(err, "unable to unmarshal JSON request")
	}
	validator, ok := processing.FromContext(r.Context())
	if ok == false {
		return it, http.StatusInternalServerError, errors.Annotatef(err, "unable to load activity validator")
	}
	validator.SetActor(f.Authenticated)

	var validateFn func(pub.Item, pub.IRI) error
	switch typ {
	case h.Outbox:
		validateFn = validator.ValidateClientActivity
	case h.Inbox:
		validateFn = validator.ValidateServerActivity
	default:
		return it, http.StatusNotAcceptable, errors.NewMethodNotAllowed(err, "Collection %s does not receive Activity requests", typ)
	}
	if err = validateFn(it, f.IRI); err != nil {
		return it, http.StatusNotAcceptable, err
	}

	err = pub.OnActivity(it, func(a *pub.Activity) error {
		// TODO(marius): this should be handled in the processing package
		if a.AttributedTo == nil {
			a.AttributedTo = f.Authenticated
		}

		it, err = processing.ProcessActivity(repo, a, typ)
		if err != nil {
			return errors.Annotatef(err, "Can't save activity %s to %s", it.GetType(), f.Collection)
		}
		if typ == h.Outbox {
			// C2S - get recipients and cleanup activity
			recipients := a.Recipients()
			func(rec pub.ItemCollection) {
				// TODO(marius): for C2S activities propagate them
			}(recipients)
		}
		return nil
	})
	if err != nil {
		return it, http.StatusInternalServerError, errors.Annotatef(err, "Unable to process %s activity", it.GetType())
	}

	status := http.StatusOK
	if it.GetType() == pub.DeleteType {
		status = http.StatusGone
	}
	if it.GetType() == pub.CreateType {
		status = http.StatusCreated
	}

	return it, status, nil
}

// HandleItem serves content from the following, followers, liked, and likes end-points
// that returns a single ActivityPub object
func HandleItem(r *http.Request, repo storage.ObjectLoader) (pub.Item, error) {
	collection := h.Typer.Type(r)

	var items pub.ItemCollection
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
		f.IRI = pub.IRI(iri)
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
		// it's the service actor
		if actLoader, ok := repo.(storage.ActorLoader); ok {
			items, _, err = actLoader.LoadActors(f)
		}
		if len(items) == 0 {
			if saver, ok := repo.(st.CanBootstrap); ok {
				service := activitypub.Self(activitypub.DefaultServiceIRI(f.IRI.String()))
				err := saver.CreateService(service)
				if err != nil {
					return nil, err
				}
				items = pub.ItemCollection{service}
			}
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

func loadItem(items pub.ItemCollection, f activitypub.Paginator, baseURL string) (pub.Item, error) {
	return items.First(), nil
}
