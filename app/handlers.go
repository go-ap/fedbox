package app

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"strings"

	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/client"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	st "github.com/go-ap/fedbox/storage"
	h "github.com/go-ap/handlers"
	"github.com/go-ap/processing"
	"github.com/go-ap/storage"
)

type pathTyper struct{}

func (d pathTyper) Type(r *http.Request) h.CollectionType {
	if r.URL == nil || len(r.URL.Path) == 0 {
		return h.Unknown
	}
	col := h.Unknown
	pathElements := strings.Split(r.URL.Path[1:], "/") // Skip first /
	for i := len(pathElements) - 1; i >= 0; i-- {
		col = h.CollectionType(strings.ToLower(pathElements[i]))
		if h.ActivityPubCollections.Contains(col) {
			return col
		}
		if ap.FedboxCollections.Contains(col) {
			return col
		}
	}

	return col
}

func reqURL(r *http.Request) string {
	scheme := "http"
	if Config.Secure || r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL.Path)
}

// HandleCollection serves content from the generic collection end-points
// that return ActivityPub objects or activities
func HandleCollection(fb FedBOX) h.CollectionHandlerFn {
	return func(typ h.CollectionType, r *http.Request, repo storage.CollectionLoader) (pub.CollectionInterface, error) {
		var col pub.CollectionInterface

		f, err := ap.FromRequest(r)
		if err != nil {
			return nil, errors.NewNotValid(err, "unable to load filters from request")
		}
		ap.LoadCollectionFilters(r, f)
		if !ap.ValidCollection(typ) {
			return nil, errors.NotFoundf("collection '%s' not found", f.Collection)
		}

		col, err = repo.LoadCollection(f)
		if err != nil {
			return nil, err
		}
		col, err = ap.PaginateCollection(col, f)
		for _, it := range col.Collection() {
			// Remove bcc and bto - probably should be moved to a different place
			// TODO(marius): move this to the go-ap/activtiypub helpers: CleanRecipients(Item)
			if s, ok := it.(pub.HasRecipients); ok {
				s.Clean()
			}
		}
		return col, err
	}
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
func HandleRequest(fb FedBOX) h.ActivityHandlerFn {
	errLogger := client.LogFn(fb.errFn)
	infoLogger := client.LogFn(fb.infFn)

	return func(typ h.CollectionType, r *http.Request, repo storage.Repository) (pub.Item, int, error) {
		var it pub.Item

		f, err := ap.FromRequest(r)
		if err != nil {
			return it, 0, errors.NewNotValid(err, "unable to load filters from request")
		}
		ap.LoadCollectionFilters(r, f)

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

		processor, validator, err := processing.New(
			processing.SetIRI(pub.IRI(Config.BaseURL)),
			processing.SetClient(client.New(
				client.SetInfoLogger(infoLogger),
				client.SetErrorLogger(errLogger),
			)),
			processing.SetStorage(repo),
			processing.SetInfoLogger(infoLogger),
			processing.SetErrorLogger(errLogger),
		)
		if err != nil {
			return it, http.StatusInternalServerError, errors.NewNotValid(err, "unable to initialize validator and processor")
		}
		validator.SetActor(f.Authenticated)

		var validateFn func(pub.Item, pub.IRI) error
		var processFn func(pub.Item) (pub.Item, error)
		switch typ {
		case h.Outbox:
			validateFn = validator.ValidateClientActivity
			processFn = processor.ProcessClientActivity
		case h.Inbox:
			validateFn = validator.ValidateServerActivity
			processFn = func(a pub.Item) (pub.Item, error) {
				return a, errors.NotImplementedf("S2S activities not implemented")
			}
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

			it, err = processFn(a)
			if err != nil {
				return errors.Annotatef(err, "Can't save activity %s to %s", it.GetType(), f.Collection)
			}
			return nil
		})
		if err != nil {
			return it, http.StatusInternalServerError, err
		}

		status := http.StatusCreated
		if it.GetType() == pub.DeleteType {
			status = http.StatusGone
		}

		return it, status, nil
	}
}

// HandleItem serves content from the following, followers, liked, and likes end-points
// that returns a single ActivityPub object
func HandleItem(fb FedBOX) h.ItemHandlerFn {
	return func(r *http.Request, repo storage.ObjectLoader) (pub.Item, error) {
		collection := h.Typer.Type(r)

		var items pub.ItemCollection
		f, err := ap.FromRequest(r)

		where := ""
		what := ""
		if len(collection) > 0 {
			where = fmt.Sprintf(" in %s", collection)
		}
		if err != nil {
			return nil, errors.NotFoundf("%snot found", what)
		}
		ap.LoadItemFilters(r, f)

		iri := reqURL(r)
		if len(f.IRI) == 0 {
			f.IRI = pub.IRI(iri)
		}
		what = fmt.Sprintf("%s ", path.Base(iri))
		f.MaxItems = 1

		if ap.ValidCollection(f.Collection) {
			if f.Collection == ap.ActorsType {
				if actLoader, ok := repo.(storage.ActorLoader); ok {
					items, _, err = actLoader.LoadActors(f)
				}
			} else if ap.ValidActivityCollection(f.Collection) {
				if actLoader, ok := repo.(storage.ActivityLoader); ok {
					items, _, err = actLoader.LoadActivities(f)
				}
			} else {
				items, _, err = repo.LoadObjects(f)
			}
		} else if f.Collection == "" {
			// it's the service actor
			if actLoader, ok := repo.(storage.ActorLoader); ok {
				items, _, err = actLoader.LoadActors(f)
			}
			if len(items) == 0 {
				if saver, ok := repo.(st.CanBootstrap); ok {
					service := ap.Self(ap.DefaultServiceIRI(f.IRI.String()))
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
		for _, it := range items {
			// Remove bcc and bto - probably should be moved to a different place
			if s, ok := it.(pub.HasRecipients); ok {
				s.Clean()
			}
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
}

func loadItem(items pub.ItemCollection, f ap.Paginator, baseURL string) (pub.Item, error) {
	return items.First(), nil
}
