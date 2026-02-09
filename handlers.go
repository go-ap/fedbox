package fedbox

import (
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"git.sr.ht/~mariusor/lw"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/cache"
	"github.com/go-ap/client"
	"github.com/go-ap/errors"
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

func reqURL(r http.Request, secure bool) string {
	scheme := "http"
	if secure || r.TLS != nil {
		scheme = "https"
	}
	u := url.URL{
		Scheme:   scheme,
		Host:     r.Host,
		Path:     r.URL.Path,
		RawPath:  r.URL.RawPath,
		RawQuery: r.URL.RawQuery,
	}
	u.Scheme = scheme
	u.Host = r.Host
	return u.String()
}

func FedBOXClient(fb *FedBOX) *client.C {
	return ActorClient(fb.Base, fb.Service.ID)
}

// HandleCollection serves content from the generic collection end-points
// that return ActivityPub objects or activities
func HandleCollection(fb *FedBOX) processing.CollectionHandlerFn {
	if fb == nil {
		return outOfOrderCollectionHandler
	}
	return func(typ vocab.CollectionPath, r *http.Request) (vocab.CollectionInterface, error) {
		if typ == vocab.Unknown {
			return nil, errors.NotFoundf("%s not found", r.URL.Path)
		}
		if !filters.ValidCollection(typ) {
			return nil, errors.NotFoundf("collection '%s' not found", typ)
		}

		// NOTE(marius): this is the main collection page, let's redirect to its first page.
		//
		// This would avoid clients having to parse the first page twice when
		// iterating through a collection:
		// * once the main collection page (which doesn't have a Next property, but just First)
		// * the second as the first page of the collection (which has a link to the Next page)
		q := r.URL.Query()
		if filters.PaginatorValues(q).Count() < 0 {
			for k, vv := range filters.FirstPage() {
				q[k] = vv
			}
			r.URL.RawQuery = q.Encode()
			return nil, errors.SeeOther(r.URL.String())
		}

		colUrl := reqURL(*r, fb.Conf.Secure)
		iri := vocab.IRI(colUrl)
		authorized := fb.actorFromRequestWithClient(r, FedBOXClient(fb), iri)
		cacheKey := CacheKey(fb, authorized, *r)

		it := fb.caches.Load(cacheKey)
		fromCache := !vocab.IsNil(it)

		var err error
		if !fromCache {
			fil := make(filters.Checks, 0)
			// NOTE(marius): I want a way to make that the owner of a collection would automatically
			// be authorized for all objects inside _even_ when they don't appear in the recipients list.
			//
			// Until that behaviour can be added to the filters module: https://todo.sr.ht/~mariusor/go-activitypub/433
			// we can remove the authorization check if actor extracted from the authorization header
			// matches the owner of the collection.
			_, maybeCol := vocab.Split(iri)
			if col := maybeCol.Of(authorized); vocab.IsNil(col) || col.GetLink().Equal(iri) {
				fil = append(fil, filters.Authorized(authorized.ID))
			}
			fil = append(fil, filters.FromValues(r.URL.Query())...)

			repo := fb.Storage
			if it, err = repo.Load(iri, fil...); err != nil {
				return nil, err
			}
		}
		if vocab.IsNil(it) || !it.IsCollection() {
			return nil, errors.NotFoundf("%s not found", typ)
		}

		var col vocab.CollectionInterface
		err = vocab.OnCollectionIntf(it, func(c vocab.CollectionInterface) error {
			col = c
			return nil
		})
		if err != nil {
			return nil, err
		}

		vocab.CleanRecipients(col)
		for _, ob := range col.Collection() {
			// Remove bcc and bto
			vocab.CleanRecipients(ob)
		}

		if !fromCache {
			fb.caches.Store(cacheKey, col)
		} else {
			// NOTE(marius): signal the [processing.CollectionHandlerFn]
			// that the collection was loaded from cache.
			err = processing.NotModified
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

// CacheKey generates a unique vocab.IRI hash based on its authenticated user and other parameters
func CacheKey(fb *FedBOX, auth vocab.Actor, r http.Request) vocab.IRI {
	u := r.URL
	if !auth.ID.Equal(vocab.PublicNS) {
		u.User = url.User(filepath.Base(auth.ID.String()))
	}
	r.URL = u
	return vocab.IRI(reqURL(r, fb.Conf.Secure))
}

// HandleActivity handles POST requests to an ActivityPub actor's inbox/outbox, based on the CollectionType
func HandleActivity(fb *FedBOX) processing.ActivityHandlerFn {
	if fb == nil {
		return outOfOrderActivityHandler
	}

	return func(receivedIn vocab.IRI, r *http.Request) (vocab.Item, int, error) {
		var it vocab.Item

		if ok, err := ValidateRequest(r); !ok {
			fb.errFn("failed request validation: %+s", err)
			return it, errors.HttpStatus(err), err
		}
		body, err := io.ReadAll(r.Body)
		if err != nil || len(body) == 0 {
			fb.errFn("failed loading body: %+s", err)
			return it, http.StatusBadRequest, errors.NewNotValid(err, "unable to read request body")
		}

		if it, err = vocab.UnmarshalJSON(body); err != nil {
			fb.errFn("failed unmarshalling jsonld body: %+s", err)
			return it, http.StatusInternalServerError, errors.NewNotValid(err, "unable to unmarshal JSON request")
		}
		if vocab.IsNil(it) {
			return it, http.StatusInternalServerError, errors.NewNotValid(err, "unable to unmarshal JSON request")
		}

		l := fb.Logger.WithContext(lw.Ctx{"log": "processing"})

		authorized := fb.actorFromRequestWithClient(r, ActorClient(fb.Base, vocab.PublicNS), receivedIn)
		if authorized.ID.Equal(vocab.PublicNS) {
			fb.errFn("invalid Anonymous actor request: %s", receivedIn)
			return it, http.StatusUnauthorized, errors.Unauthorizedf("authorized Actor is invalid")
		}

		repo := fb.Storage

		baseIRI := vocab.IRI(fb.Conf.BaseURL)
		initFns := make([]processing.OptionFn, 0)
		initFns = append(initFns,
			processing.WithIRI(baseIRI, InternalIRI),
			processing.WithClient(ActorClient(fb.Base, receivedIn)),
			processing.WithStorage(repo),
			processing.WithLogger(l),
			processing.WithIDGenerator(GenerateID(baseIRI)),
		)
		if fb.keyGenerator != nil {
			initFns = append(initFns, processing.WithActorKeyGenerator(fb.keyGenerator))
		}
		if !fb.Conf.Env.IsTest() {
			initFns = append(initFns, processing.Async)
		}
		processor := processing.New(initFns...)
		if err != nil {
			fb.errFn("failed initializing the Activity processor: %+s", err)
			return it, http.StatusInternalServerError, errors.NewNotValid(err, "unable to initialize processor")
		}

		typ := it.GetType()
		if it, err = processor.ProcessActivity(it, authorized, receivedIn); err != nil {
			fb.errFn("failed processing activity: %+s", err)
			return it, errors.HttpStatus(err), errors.Annotatef(err, "Unable to save activity %s to %s", typ, receivedIn)
		}
		_ = vocab.OnActivity(it, func(act *vocab.Activity) error {
			if err := cache.ActivityPurge(fb.caches, act, receivedIn); err != nil {
				fb.errFn("unable to purge cache: %+s", err)
			}
			return nil
		})

		status := http.StatusCreated
		if vocab.DeleteType.Match(it.GetType()) {
			status = http.StatusGone
		}

		fb.infFn("All OK!")
		return it, status, nil
	}
}

// HandleItem serves content from the following, followers, liked, and likes end-points
// that returns a single ActivityPub object
func HandleItem(fb *FedBOX) processing.ItemHandlerFn {
	if fb == nil {
		return outOfOrderItemHandler
	}
	return func(r *http.Request) (vocab.Item, error) {
		iri := vocab.IRI(reqURL(*r, fb.Conf.Secure))

		authorized := fb.actorFromRequestWithClient(r, ActorClient(fb.Base, vocab.PublicNS), iri)
		cacheKey := CacheKey(fb, authorized, *r)

		it := fb.caches.Load(cacheKey)
		fromCache := !vocab.IsNil(it)

		what := r.URL.Path

		if !fromCache {
			repo := fb.Storage
			var err error
			var f filters.Check
			f = filters.Authorized(authorized.ID)
			if it, err = repo.Load(iri, f); err != nil {
				return nil, errors.NotFoundf("%s was not found", what)
			}
		}
		var err error
		if vocab.IsItemCollection(it) {
			err = vocab.OnCollectionIntf(it, func(col vocab.CollectionInterface) error {
				if col.Count() == 0 {
					return errors.NotFoundf("%s not found", what)
				}

				if col.Count() > 1 {
					return errors.Conflictf("Too many %s found", what)
				}
				it = col.Collection().First()
				return nil
			})
			if err != nil {
				return nil, err
			}
		}

		if !fromCache {
			fb.caches.Store(cacheKey, it)
		} else {
			// NOTE(marius): signal the [processing.CollectionHandlerFn]
			// that the collection was loaded from cache.
			err = processing.NotModified
		}

		// Remove bcc and bto
		return vocab.CleanRecipients(it), err
	}
}

// ProxyURL
// Endpoint URI so this actor's clients may access remote ActivityStreams objects which require authentication to access.
// To use this endpoint, the client posts an x-www-form-urlencoded id parameter with the value being the id of the
// requested ActivityStreams object.
//
// https://www.w3.org/TR/activitypub/#proxyUrl
func ProxyURL(fb *FedBOX) http.Handler {
	if fb == nil {
		return processing.ItemHandlerFn(outOfOrderItemHandler)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			errors.HandleError(errors.UnsupportedMediaTypef("content type is not supported by the proxy")).ServeHTTP(w, r)
			return
		}
		id := r.FormValue("id")
		if id == "" {
			errors.HandleError(errors.NotFoundf(`invalid 'id' value for proxy retrieval`)).ServeHTTP(w, r)
			return
		}

		// NOTE(marius): if we can load a valid actor from the request, we use it for fetching the/
		// remote resource pointed at by "id"
		authorized := fb.actorFromRequestWithClient(r, ActorClient(fb.Base, vocab.PublicNS), vocab.IRI(id))
		cl := ActorClient(fb.Base, authorized)
		res, err := cl.Get(id)
		if err != nil {
			errors.HandleError(errors.NotFoundf(`invalid 'id' value for proxy retrieval`)).ServeHTTP(w, r)
			return
		}
		defer res.Body.Close()
		fb.Logger.WithContext(lw.Ctx{"iri": id, "actor": authorized.ID, "status": res.Status}).Infof("request proxied successfully")

		w.WriteHeader(res.StatusCode)
		for k := range res.Header {
			w.Header().Set(k, res.Header.Get(k))
		}
		_, _ = io.Copy(w, res.Body)
	})
}
