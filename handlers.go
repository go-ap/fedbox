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
	ap "github.com/go-ap/fedbox/activitypub"
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

// HandleCollection serves content from the generic collection end-points
// that return ActivityPub objects or activities
func HandleCollection(fb FedBOX) processing.CollectionHandlerFn {
	return func(typ vocab.CollectionPath, r *http.Request) (vocab.CollectionInterface, error) {
		if typ == vocab.Unknown {
			return nil, errors.NotFoundf("%s not found", r.URL.Path)
		}
		if !filters.ValidCollection(typ) {
			return nil, errors.NotFoundf("collection '%s' not found", typ)
		}

		repo := fb.storage

		iri := vocab.IRI(reqURL(*r, fb.conf.Secure))
		cacheKey := CacheKey(fb, *r)

		it := fb.caches.Load(cacheKey)
		fromCache := !vocab.IsNil(it)

		var err error
		if !fromCache {
			fil := filters.Checks{filters.Authorized(fb.actorFromRequest(r).ID)}
			fil = append(fil, filters.FromValues(r.URL.Query())...)

			if it, err = repo.Load(iri, fil...); err != nil {
				return nil, err
			}
		}
		if vocab.IsNil(it) || !it.IsCollection() {
			return nil, errors.NotFoundf("%s not found", typ)
		}

		if !fromCache {
			fb.caches.Store(cacheKey, it)
		}

		var col vocab.CollectionInterface
		err = vocab.OnCollectionIntf(it, func(c vocab.CollectionInterface) error {
			for _, it := range c.Collection() {
				// Remove bcc and bto - probably should be moved to a different place
				// TODO(marius): move this to the go-ap/activtiypub helpers: CleanRecipients(Item)
				if s, ok := it.(vocab.HasRecipients); ok {
					s.Clean()
				}
			}
			col = c
			return nil
		})
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

// CacheKey generates a unique vocab.IRI hash based on its authenticated user and other parameters
func CacheKey(fb FedBOX, r http.Request) vocab.IRI {
	u := r.URL
	if auth := fb.actorFromRequest(&r); !auth.ID.Equals(vocab.PublicNS, true) {
		u.User = url.User(filepath.Base(auth.ID.String()))
	}
	r.URL = u
	return vocab.IRI(reqURL(r, fb.conf.Secure))
}

// HandleActivity handles POST requests to an ActivityPub actor's inbox/outbox, based on the CollectionType
func HandleActivity(fb FedBOX) processing.ActivityHandlerFn {
	return func(receivedIn vocab.IRI, r *http.Request) (vocab.Item, int, error) {
		repo := fb.storage
		var it vocab.Item
		fb.infFn("received req %s: %s", r.Method, r.RequestURI)

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
			fb.errFn("failed unmarshalling jsonld body: %+s", err)
			return it, http.StatusInternalServerError, errors.NewNotValid(err, "unable to unmarshal JSON request")
		}
		if vocab.IsNil(it) {
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

		// NOTE(marius): this probably leaks the actor in requests we don't want it in
		// The solution is to move the auth and client objects into the request scope,
		// instead of storing them on the top level FedBOX object.
		fb.client.SignFn(s2sSignFn(fb.self, fb.storage, fb.logger))
		auth := fb.actorFromRequest(r)
		processor.SetActor(&auth)

		if fb.keyGenerator != nil {
			processing.WithActorKeyGenerator(fb.keyGenerator)
		}

		if it, err = processor.ProcessActivity(it, receivedIn); err != nil {
			fb.errFn("failed processing activity: %+s", err)
			return it, errors.HttpStatus(err), errors.Annotatef(err, "Can't save activity %s to %s", it.GetType(), filepath.Base(r.URL.Path))
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

		iri := vocab.IRI(reqURL(*r, fb.conf.Secure))

		cacheKey := CacheKey(fb, *r)

		it := fb.caches.Load(cacheKey)
		fromCache := !vocab.IsNil(it)

		what := r.URL.Path

		var err error
		if !fromCache {
			var f filters.Check
			f = filters.Authorized(fb.actorFromRequest(r).ID)
			if it, err = repo.Load(iri, f); err != nil {
				return nil, errors.NotFoundf("%s was not found", what)
			}
		}
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
		}

		if s, ok := it.(vocab.HasRecipients); ok {
			// Remove bcc and bto - probably should be moved to a different place
			s.Clean()
		}
		return it, nil
	}
}
