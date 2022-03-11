package app

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"

	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/client"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/cache"
	st "github.com/go-ap/fedbox/storage"
	h "github.com/go-ap/handlers"
	"github.com/go-ap/processing"
	"github.com/go-ap/storage"
)

type pathTyper struct{}

func (d pathTyper) Type(r *http.Request) h.CollectionType {
	col := h.Unknown
	if r.URL == nil || len(r.URL.Path) == 0 {
		return col
	}

	pathElements := strings.Split(r.URL.Path[1:], "/") // Skip first /
	for i := len(pathElements) - 1; i >= 0; i-- {
		col = h.CollectionType(strings.ToLower(pathElements[i]))
		if h.ActivityPubCollections.Contains(col) {
			return col
		}
		if ap.FedBOXCollections.Contains(col) {
			return col
		}
	}
	return col
}

func reqURL(r *http.Request, secure bool) string {
	scheme := "http"
	if secure || r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s%s", scheme, r.Host, r.RequestURI)
}

func orderItems(col pub.ItemCollection) pub.ItemCollection {
	sort.SliceStable(col, func(i, j int) bool {
		return pub.ItemOrderTimestamp(col[i], col[j])
	})
	return col
}

// HandleCollection serves content from the generic collection end-points
// that return ActivityPub objects or activities
func HandleCollection(fb FedBOX) h.CollectionHandlerFn {
	return func(typ h.CollectionType, r *http.Request, repo storage.ReadStore) (pub.CollectionInterface, error) {
		if !ap.ValidCollection(typ) {
			return nil, errors.NotFoundf("collection '%s' not found", typ)
		}

		f, err := ap.FromRequest(r, fb.Config().BaseURL)
		if err != nil {
			return nil, errors.NewNotValid(err, "unable to load filters from request")
		}
		ap.LoadCollectionFilters(r, f)

		cacheKey := ap.CacheKey(f)
		it := fb.caches.Get(cacheKey)
		fromCache := !pub.IsNil(it)

		if !fromCache {
			if it, err = repo.Load(f.GetLink()); err != nil {
				return nil, err
			}
		}
		if !it.IsCollection() {
			return nil, errors.NotFoundf("collection '%s' not found", f.Collection)
		}

		c := new(pub.OrderedCollection)
		c.Type = pub.OrderedCollectionType
		err = pub.OnCollectionIntf(it, func(items pub.CollectionInterface) error {
			ff := *f
			ff.Authenticated = nil
			c.ID = ff.GetLink()
			c.OrderedItems = orderItems(items.Collection())
			c.TotalItems = c.OrderedItems.Count()
			return nil
		})
		if err != nil {
			return nil, err
		}

		var toStore pub.OrderedCollection
		if !fromCache && c.Count() > 0 {
			toStore = *c
		}
		var col pub.CollectionInterface = c
		if col, err = ap.PaginateCollection(col, f); err != nil {
			return nil, err
		}
		if !fromCache && toStore.Collection() != nil {
			fb.caches.Set(cacheKey, toStore)
		}
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

// GenerateID
func GenerateID(base pub.IRI) func(it pub.Item, col pub.Item, by pub.Item) (pub.ID, error) {
	return func(it pub.Item, col pub.Item, by pub.Item) (pub.ID, error) {
		typ := it.GetType()

		var partOf pub.IRI
		if pub.ActivityTypes.Contains(typ) || pub.IntransitiveActivityTypes.Contains(typ) {
			partOf = ap.ActivitiesType.IRI(base)
		} else if pub.ActorTypes.Contains(typ) || typ == pub.ActorType {
			partOf = ap.ActorsType.IRI(base)
		} else {
			partOf = ap.ObjectsType.IRI(base)
		}
		return ap.GenerateID(it, partOf, by)
	}
}

// HandleRequest handles POST requests to an ActivityPub To's inbox/outbox, based on the CollectionType
func HandleRequest(fb FedBOX) h.ActivityHandlerFn {
	errLogger := client.LogFn(fb.errFn)
	infoLogger := client.LogFn(fb.infFn)
	clientErrLogger := func(...client.Ctx) client.LogFn {
		return errLogger
	}
	clientInfoLogger := func(...client.Ctx) client.LogFn {
		return infoLogger
	}
	return func(typ h.CollectionType, r *http.Request, repo storage.Store) (pub.Item, int, error) {
		var it pub.Item

		f, err := ap.FromRequest(r, fb.Config().BaseURL)
		if err != nil {
			return it, 0, errors.NewNotValid(err, "unable to load filters from request")
		}
		ap.LoadCollectionFilters(r, f)

		if ok, err := ValidateRequest(r); !ok {
			return it, http.StatusInternalServerError, errors.NewNotValid(err, "unrecognized ActivityPub content type")
		}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil || len(body) == 0 {
			return it, http.StatusInternalServerError, errors.NewNotValid(err, "unable to read request body")
		}
		if it, err = pub.UnmarshalJSON(body); err != nil {
			return it, http.StatusInternalServerError, errors.NewNotValid(err, "unable to unmarshal JSON request")
		}

		baseIRI := pub.IRI(fb.Config().BaseURL)
		processor, validator, err := processing.New(
			processing.SetIRI(baseIRI, InternalIRI),
			processing.SetClient(client.New(
				client.SetInfoLogger(clientInfoLogger),
				client.SetErrorLogger(clientErrLogger),
				client.SkipTLSValidation(!fb.Config().Env.IsProd()),
			)),
			processing.SetStorage(repo),
			processing.SetInfoLogger(infoLogger),
			processing.SetErrorLogger(errLogger),
			processing.SetIDGenerator(GenerateID(baseIRI)),
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
			processFn = processor.ProcessServerActivity
		default:
			return it, http.StatusNotAcceptable, errors.NewMethodNotAllowed(err, "Collection %s does not receive Activity requests", typ)
		}
		if err = validateFn(it, f.IRI); err != nil {
			return it, http.StatusNotAcceptable, err
		}
		pub.OnActivity(it, func(a *pub.Activity) error {
			// TODO(marius): this should be handled in the processing package
			if a.AttributedTo == nil {
				a.AttributedTo = f.Authenticated
			}
			return nil
		})
		if it, err = processFn(it); err != nil {
			return it, errors.HttpStatus(err), errors.Annotatef(err, "Can't save activity %s to %s", it.GetType(), f.Collection)
		}
		err = pub.OnActivity(it, func(act *pub.Activity) error {
			return cache.ActivityPurge(fb.caches, act, typ)
		})
		if err != nil {
			infoLogger("unable to purge cache: %s", err)
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
	return func(r *http.Request, repo storage.ReadStore) (pub.Item, error) {
		collection := h.Typer.Type(r)

		f, err := ap.FromRequest(r, fb.Config().BaseURL)
		if err != nil {
			return nil, errors.NewNotValid(err, "unable to load filters from request")
		}
		ap.LoadItemFilters(r, f)

		cacheKey := ap.CacheKey(f)
		it := fb.caches.Get(cacheKey)
		fromCache := !pub.IsNil(it)

		iri := reqURL(r, fb.Config().Secure)
		if len(f.IRI) == 0 {
			f.IRI = pub.IRI(iri)
		}

		where := ""
		what := ""
		if len(collection) > 0 {
			where = fmt.Sprintf(" in %s", collection)
		}
		what = fmt.Sprintf("%s ", path.Base(iri))
		if u, err := url.ParseRequestURI(iri); err == nil {
			what = fmt.Sprintf("%s ", path.Base(u.Path))
		}

		f.MaxItems = 1

		if len(f.Collection) > 0 && !ap.ValidCollection(f.Collection) {
			return nil, errors.NotFoundf("%s not found", r.URL.Path)
		}
		if !fromCache {
			if it, err = repo.Load(f.GetLink()); err != nil {
				return nil, err
			}
		}
		var items pub.ItemCollection
		if pub.IsItemCollection(it) {
			err = pub.OnCollectionIntf(it, func(col pub.CollectionInterface) error {
				items = col.Collection()
				return nil
			})
			if err != nil {
				return nil, err
			}
		} else {
			items = pub.ItemCollection{it}
		}

		if f.Collection == "" && len(items) == 0 {
			if saver, ok := repo.(st.CanBootstrap); ok {
				service := ap.Self(ap.DefaultServiceIRI(f.IRI.String()))
				if err := saver.CreateService(service); err != nil {
					return nil, err
				}
				items = pub.ItemCollection{service}
			}
		}

		if len(items) == 0 {
			return nil, errors.NotFoundf("%snot found%s", what, where)
		}

		if len(items) > 1 {
			return nil, errors.Conflictf("Too many %s found%s", what, where)
		}

		if it, err = loadItem(items, f, reqURL(r, fb.Config().Secure)); err != nil {
			return nil, errors.NotFoundf("%snot found", what)
		}

		if !fromCache {
			fb.caches.Set(cacheKey, it)
		}

		if s, ok := it.(pub.HasRecipients); ok {
			// Remove bcc and bto - probably should be moved to a different place
			s.Clean()
		}
		return it, nil
	}
}

func loadItem(items pub.ItemCollection, f ap.Paginator, baseURL string) (pub.Item, error) {
	return items.First(), nil
}
