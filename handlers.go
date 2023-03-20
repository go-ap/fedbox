package fedbox

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strings"

	"git.sr.ht/~mariusor/lw"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/client"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/cache"
	st "github.com/go-ap/fedbox/storage"
	"github.com/go-ap/filters"
	"github.com/go-ap/processing"
)

type pathTyper struct{}

var validCollections = append(filters.FedBOXCollections, vocab.ActivityPubCollections...)

func (d pathTyper) Type(r *http.Request) vocab.CollectionPath {
	col := vocab.Unknown
	if r.URL == nil || len(r.URL.Path) == 0 {
		return col
	}
	if filepath.Clean(r.URL.Path) != r.URL.Path {
		return col
	}
	pathElements := strings.Split(strings.TrimLeft(r.URL.Path, "/"), "/")
	for i := len(pathElements) - 1; i >= 0; i-- {
		piece := vocab.CollectionPath(strings.ToLower(pathElements[i]))
		if validCollections.Contains(piece) {
			col = piece
			break
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
	return func(typ vocab.CollectionPath, r *http.Request) (vocab.CollectionInterface, error) {
		repo := fb.storage
		if typ == vocab.Unknown {
			return nil, errors.NotFoundf("%s not found", r.URL.Path)
		}
		if !filters.ValidCollection(typ) {
			return nil, errors.NotFoundf("collection '%s' not found", typ)
		}

		f := filters.FromRequest(r, fb.Config().BaseURL)
		filters.LoadCollectionFilters(f, fb.actorFromRequest(r))

		cacheKey := filters.CacheKey(f)
		it := fb.caches.Get(cacheKey)
		fromCache := !vocab.IsNil(it)

		var err error
		if !fromCache {
			if it, err = repo.Load(f.GetLink()); err != nil {
				return nil, err
			}
		}
		if vocab.IsNil(it) || !it.IsCollection() {
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

var validActivityCollections = vocab.CollectionPaths{vocab.Outbox, vocab.Inbox}

func validActivityCollection(r *http.Request) bool {
	return validActivityCollections.Contains(processing.Typer.Type(r))
}

func ValidateRequest(r *http.Request) (bool, error) {
	contType := r.Header.Get("Content-Type")
	if r.Method != http.MethodPost {
		return false, errors.MethodNotAllowedf("invalid HTTP method")
	}
	if !validContentType(contType) {
		return false, errors.NotValidf("invalid content type")
	}
	if !validActivityCollection(r) {
		return false, errors.NotValidf("invalid collection")
	}

	return true, nil
}

// GenerateID creates an IRI that can be used to uniquely identify the "it" item, based on the collection "col" and
// its creator "by"
func GenerateID(base vocab.IRI) func(it vocab.Item, col vocab.Item, by vocab.Item) (vocab.ID, error) {
	return func(it vocab.Item, col vocab.Item, by vocab.Item) (vocab.ID, error) {
		typ := it.GetType()

		var partOf vocab.IRI
		if vocab.ActivityTypes.Contains(typ) || vocab.IntransitiveActivityTypes.Contains(typ) {
			partOf = filters.ActivitiesType.IRI(base)
		} else if vocab.ActorTypes.Contains(typ) || typ == vocab.ActorType {
			partOf = filters.ActorsType.IRI(base)
		} else {
			partOf = filters.ObjectsType.IRI(base)
		}
		return ap.GenerateID(it, partOf, by)
	}
}

// HandleActivity handles POST requests to an ActivityPub actor's inbox/outbox, based on the CollectionType
func HandleActivity(fb FedBOX) processing.ActivityHandlerFn {
	return func(receivedIn vocab.IRI, r *http.Request) (vocab.Item, int, error) {
		repo := fb.storage
		var it vocab.Item
		fb.infFn("received req %s: %s", r.Method, r.RequestURI)

		f := filters.FromRequest(r, fb.Config().BaseURL)
		filters.LoadCollectionFilters(f, fb.actorFromRequest(r))

		if ok, err := ValidateRequest(r); !ok {
			fb.errFn("failed request validation: %+s", err)
			return it, errors.HttpStatus(err), err
		}
		body, err := io.ReadAll(r.Body)
		if err != nil || len(body) == 0 {
			fb.errFn("failed loading body: %+s", err)
			return it, http.StatusInternalServerError, errors.NewNotValid(err, "unable to read request body")
		}
		if it, err = vocab.UnmarshalJSON(body); err != nil {
			fb.errFn("failed unmarshaling jsonld body: %+s", err)
			return it, http.StatusInternalServerError, errors.NewNotValid(err, "unable to unmarshal JSON request")
		}

		l := fb.logger.WithContext(lw.Ctx{"log": "processing"})
		baseIRI := vocab.IRI(fb.Config().BaseURL)
		processor, err := processing.New(
			processing.WithIRI(baseIRI, InternalIRI),
			processing.WithClient(&fb.client),
			processing.WithStorage(repo),
			processing.WithLogger(l),
			processing.WithIDGenerator(GenerateID(baseIRI)),
			processing.WithLocalIRIChecker(st.IsLocalIRI(repo)),
		)
		if err != nil {
			fb.errFn("failed initializing the Activity processor: %+s", err)
			return it, http.StatusInternalServerError, errors.NewNotValid(err, "unable to initialize processor")
		}
		processor.SetActor(f.Authenticated)
		if fb.keyGenerator != nil {
			processing.WithActorKeyGenerator(fb.keyGenerator)
		}

		vocab.OnActivity(it, func(a *vocab.Activity) error {
			// TODO(marius): this should be handled in the processing package
			if a.AttributedTo == nil {
				a.AttributedTo = f.Authenticated
			}
			return nil
		})
		if it, err = processor.ProcessActivity(it, receivedIn); err != nil {
			fb.errFn("failed processing activity: %+s", err)
			return it, errors.HttpStatus(err), errors.Annotatef(err, "Can't save activity %s to %s", it.GetType(), f.Collection)
		}
		err = vocab.OnActivity(it, func(act *vocab.Activity) error {
			return cache.ActivityPurge(fb.caches, act, receivedIn)
		})
		if err != nil {
			fb.errFn("unable to purge cache: %+s", err)
		}

		status := http.StatusCreated
		if it.GetType() == vocab.DeleteType {
			status = http.StatusGone
		}

		fb.infFn("All OK!")
		return it, status, nil
	}
}

// HandleItem serves content from the following, followers, liked, and likes end-points
// that returns a single ActivityPub object
func HandleItem(fb FedBOX) processing.ItemHandlerFn {
	return func(r *http.Request) (vocab.Item, error) {
		repo := fb.storage
		f := filters.FromRequest(r, fb.Config().BaseURL)
		if !f.IRI.Equals(fb.self.GetLink(), true) && !filters.ValidCollection(f.Collection) {
			return nil, errors.NotFoundf("%s not found", r.URL.Path)
		}

		filters.LoadItemFilters(f, fb.actorFromRequest(r))

		cacheKey := filters.CacheKey(f)
		it := fb.caches.Get(cacheKey)
		fromCache := !vocab.IsNil(it)

		iri := reqURL(r, fb.Config().Secure)
		if len(f.IRI) == 0 {
			f.IRI = vocab.IRI(iri)
		}

		where := ""
		what := ""
		if u, err := url.ParseRequestURI(iri); err == nil {
			what = fmt.Sprintf("%s ", u.Path)
		}
		if len(f.Collection) > 0 {
			where = fmt.Sprintf(" in %s", f.Collection)
			what = fmt.Sprintf("%s ", strings.Replace(what, string(f.Collection), "", 1))
		}

		f.MaxItems = 1

		var err error
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
