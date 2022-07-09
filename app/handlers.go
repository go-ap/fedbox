package app

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/client"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/cache"
	st "github.com/go-ap/fedbox/storage"
	"github.com/go-ap/processing"
)

type pathTyper struct{}

func (d pathTyper) Type(r *http.Request) vocab.CollectionPath {
	col := vocab.Unknown
	if r.URL == nil || len(r.URL.Path) == 0 {
		return col
	}

	pathElements := strings.Split(r.URL.Path[1:], "/") // Skip first /
	for i := len(pathElements) - 1; i >= 0; i-- {
		col = vocab.CollectionPath(strings.ToLower(pathElements[i]))
		if vocab.ActivityPubCollections.Contains(col) {
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

func orderItems(col vocab.ItemCollection) vocab.ItemCollection {
	sort.SliceStable(col, func(i, j int) bool {
		return vocab.ItemOrderTimestamp(col[i], col[j])
	})
	return col
}

// HandleCollection serves content from the generic collection end-points
// that return ActivityPub objects or activities
func HandleCollection(fb FedBOX) processing.CollectionHandlerFn {
	return func(typ vocab.CollectionPath, r *http.Request, repo processing.ReadStore) (vocab.CollectionInterface, error) {
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
		fromCache := !vocab.IsNil(it)

		if !fromCache {
			if it, err = repo.Load(f.GetLink()); err != nil {
				return nil, err
			}
		}
		if !it.IsCollection() {
			return nil, errors.NotFoundf("collection '%s' not found", f.Collection)
		}

		c := new(vocab.OrderedCollection)
		c.Type = vocab.OrderedCollectionType
		err = vocab.OnCollectionIntf(it, func(items vocab.CollectionInterface) error {
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

		var toStore vocab.OrderedCollection
		if !fromCache && c.Count() > 0 {
			toStore = *c
		}
		var col vocab.CollectionInterface = c
		if col, err = ap.PaginateCollection(col, f); err != nil {
			return nil, err
		}
		if !fromCache && toStore.Collection() != nil {
			fb.caches.Set(cacheKey, toStore)
		}
		for _, it := range col.Collection() {
			// Remove bcc and bto - probably should be moved to a different place
			// TODO(marius): move this to the go-ap/activtiypub helpers: CleanRecipients(Item)
			if s, ok := it.(vocab.HasRecipients); ok {
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

// GenerateID creates an IRI that can be used to uniquely identify the "it" item, based on the collection "col" and
// its creator "by"
func GenerateID(base vocab.IRI) func(it vocab.Item, col vocab.Item, by vocab.Item) (vocab.ID, error) {
	return func(it vocab.Item, col vocab.Item, by vocab.Item) (vocab.ID, error) {
		typ := it.GetType()

		var partOf vocab.IRI
		if vocab.ActivityTypes.Contains(typ) || vocab.IntransitiveActivityTypes.Contains(typ) {
			partOf = ap.ActivitiesType.IRI(base)
		} else if vocab.ActorTypes.Contains(typ) || typ == vocab.ActorType {
			partOf = ap.ActorsType.IRI(base)
		} else {
			partOf = ap.ObjectsType.IRI(base)
		}
		return ap.GenerateID(it, partOf, by)
	}
}

// HandleRequest handles POST requests to an ActivityPub To's inbox/outbox, based on the CollectionType
func HandleRequest(fb FedBOX) processing.ActivityHandlerFn {
	errLogger := client.LogFn(fb.errFn)
	infoLogger := client.LogFn(fb.infFn)
	clientErrLogger := func(...client.Ctx) client.LogFn {
		return errLogger
	}
	clientInfoLogger := func(...client.Ctx) client.LogFn {
		return infoLogger
	}
	return func(receivedIn vocab.IRI, r *http.Request, repo processing.Store) (vocab.Item, int, error) {
		var it vocab.Item

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
		if it, err = vocab.UnmarshalJSON(body); err != nil {
			return it, http.StatusInternalServerError, errors.NewNotValid(err, "unable to unmarshal JSON request")
		}

		baseIRI := vocab.IRI(fb.Config().BaseURL)
		processor, err := processing.New(
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
		processor.SetActor(f.Authenticated)
		if metaSaver, ok := repo.(st.MetadataTyper); ok {
			processing.SetActorKeyGenerator(AddKeyToPerson(metaSaver))
		}

		if err = processor.ValidateActivity(it, f.IRI); err != nil {
			return it, http.StatusNotAcceptable, err
		}
		vocab.OnActivity(it, func(a *vocab.Activity) error {
			// TODO(marius): this should be handled in the processing package
			if a.AttributedTo == nil {
				a.AttributedTo = f.Authenticated
			}
			return nil
		})
		if it, err = processor.ProcessActivity(it, receivedIn); err != nil {
			return it, errors.HttpStatus(err), errors.Annotatef(err, "Can't save activity %s to %s", it.GetType(), f.Collection)
		}
		err = vocab.OnActivity(it, func(act *vocab.Activity) error {
			return cache.ActivityPurge(fb.caches, act, receivedIn)
		})
		if err != nil {
			infoLogger("unable to purge cache: %s", err)
		}

		status := http.StatusCreated
		if it.GetType() == vocab.DeleteType {
			status = http.StatusGone
		}

		return it, status, nil
	}
}

// HandleItem serves content from the following, followers, liked, and likes end-points
// that returns a single ActivityPub object
func HandleItem(fb FedBOX) processing.ItemHandlerFn {
	return func(r *http.Request, repo processing.ReadStore) (vocab.Item, error) {
		collection := processing.Typer.Type(r)

		f, err := ap.FromRequest(r, fb.Config().BaseURL)
		if err != nil {
			return nil, errors.NewNotValid(err, "unable to load filters from request")
		}
		ap.LoadItemFilters(r, f)

		cacheKey := ap.CacheKey(f)
		it := fb.caches.Get(cacheKey)
		fromCache := !vocab.IsNil(it)

		iri := reqURL(r, fb.Config().Secure)
		if len(f.IRI) == 0 {
			f.IRI = vocab.IRI(iri)
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
		var items vocab.ItemCollection
		if vocab.IsItemCollection(it) {
			err = vocab.OnCollectionIntf(it, func(col vocab.CollectionInterface) error {
				items = col.Collection()
				return nil
			})
			if err != nil {
				return nil, err
			}
		} else {
			items = vocab.ItemCollection{it}
		}

		if f.Collection == "" && len(items) == 0 {
			if saver, ok := repo.(st.CanBootstrap); ok {
				service := ap.Self(ap.DefaultServiceIRI(f.IRI.String()))
				if err := saver.CreateService(service); err != nil {
					return nil, err
				}
				items = vocab.ItemCollection{service}
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

		if s, ok := it.(vocab.HasRecipients); ok {
			// Remove bcc and bto - probably should be moved to a different place
			s.Clean()
		}
		return it, nil
	}
}

func loadItem(items vocab.ItemCollection, f ap.Paginator, baseURL string) (vocab.Item, error) {
	return items.First(), nil
}
