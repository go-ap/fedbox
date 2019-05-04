package app

import (
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/fedbox/internal/errors"
	j "github.com/go-ap/jsonld"
	"github.com/go-chi/chi"
	"net/http"
)

func renderItem(i as.Item) ([]byte, error) {
	return j.WithContext(j.IRI(as.ActivityBaseURI)).Marshal(i)
}

// ItemHandlerFn is the type that we're using to represent handlers that return ActivityStreams
// objects. It needs to implement the http.Handler interface
type ItemHandlerFn func(http.ResponseWriter, *http.Request) (as.Item, error)

// ValidMethod validates if the current handler can process the current request
func (i ItemHandlerFn) ValidMethod( r *http.Request) bool {
	return r.Method != http.MethodGet && r.Method != http.MethodHead
}

// ServeHTTP implements the http.Handler interface for the ItemHandlerFn type
func (i ItemHandlerFn) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var dat []byte
	var status int

	if i.ValidMethod(r) {
		status = http.StatusNotAcceptable
		_, dat = errors.Render(r, errors.MethodNotAllowedf("invalid HTTP method"))
	}

	if it, err := i(w, r); err != nil {
		// HandleError
		status = http.StatusInternalServerError
		_, dat = errors.Render(r, err)
	} else {
		if dat, err = renderItem(it); err != nil {
			status = http.StatusInternalServerError
			_, dat = errors.Render(r, err)
		} else {
			status = http.StatusOK
		}
	}

	w.WriteHeader(status)
	if r.Method == http.MethodGet {
		w.Write(dat)
	}
}

// HandleActivityItem serves content from the outbox, inbox, likes, shares and replies end-points
// that returns a single ActivityPub activity
func HandleActivityItem(w http.ResponseWriter, r *http.Request) (as.Item, error) {
	collection :=  Typer.Type(r)
	repo := loader{}

	id := chi.URLParam(r, "id")

	var items as.ItemCollection
	var err error
	f := Filters{}
	f.FromRequest(r)
	f.ItemKey = []Hash{
		Hash(id),
	}
	f.MaxItems = 1

	if col := chi.URLParam(r, "collection"); len(col) > 0 {
		if CollectionType(col) == collection {
			if ValidActivityCollection(col) {
				items, err = repo.LoadActivities(f)
			} else {
				return nil, errors.BadRequestf("invalid collection %s", collection)
			}
		}
	} else {
		// Non recognized as valid collection types
		// In our case activities
		switch collection {
		case CollectionType("activities"):
			items, err = repo.LoadActivities(f)
		default:
			return nil, errors.BadRequestf("invalid collection %s", collection)
		}
	}
	if err != nil {
		return nil, err
	}
	if len(items) == 1 {
		it, err := loadItem(items, &f, reqURL(r, r.URL.Path))
		if err != nil {
			return nil, errors.NotFoundf("%s", collection)
		}
		return it, nil
	}

	return nil, errors.NotFoundf("%s %s", collection, id)
}

// HandleObjectItem serves content from the following, followers, liked, and likes end-points
// that returns a single ActivityPub object
func HandleObjectItem(w http.ResponseWriter, r *http.Request) (as.Item, error) {
	collection :=  Typer.Type(r)
	repo := loader{}

	id := chi.URLParam(r, "id")

	var items as.ItemCollection
	var err error
	f := Filters{}
	f.FromRequest(r)
	f.ItemKey = []Hash{
		Hash(id),
	}
	f.MaxItems = 1

	if col := chi.URLParam(r, "collection"); len(col) > 0 {
		if CollectionType(col) == collection {
			if ValidObjectCollection(col) {
				items, err = repo.LoadObjects(f)
			} else {
				return nil, errors.BadRequestf("invalid collection %s", collection)
			}
		}
	} else {
		// Non recognized as valid collection types
		// In our case actors and items
		switch collection {
		case CollectionType("actors"):
			items, err = repo.LoadActors(f)
		case CollectionType("items"):
			items, err = repo.LoadObjects(f)
		default:
			return nil, errors.BadRequestf("invalid collection %s", collection)
		}
	}
	if err != nil {
		return nil, err
	}
	if len(items) == 1 {
		it, err := loadItem(items, &f, reqURL(r, r.URL.Path))
		if err != nil {
			return nil, errors.NotFoundf("%s", collection)
		}
		return it, nil
	}

	return nil, errors.NotFoundf("%s %s", collection, id)
}

func loadItem(items as.ItemCollection, f Paginator, baseURL string) (as.Item, error) {
	return items[0], nil
}
