package app

import (
	h "github.com/go-ap/activitypub/handler"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/fedbox/internal/errors"
	st "github.com/go-ap/fedbox/storage"
	"github.com/go-chi/chi"
	"net/http"
)

// HandleActivityItem serves content from the outbox, inbox, likes, shares and replies end-points
// that returns a single ActivityPub activity
func HandleActivityItem(w http.ResponseWriter, r *http.Request) (as.Item, error) {
	collection := h.Typer.Type(r)
	repo := st.Loader{}

	id := chi.URLParam(r, "id")

	var items as.ItemCollection
	var err error
	f := &st.Filters{}
	f.FromRequest(r)
	f.ItemKey = []st.Hash{
		st.Hash(id),
	}
	f.MaxItems = 1

	if h.ValidActivityCollection(string(f.Collection)) {
		items, err = repo.LoadActivities(f)
	} else {
		// Non recognized as valid collection types
		// In our case activities
		switch collection {
		case h.CollectionType("activities"):
			items, err = repo.LoadActivities(f)
		default:
			return nil, errors.BadRequestf("invalid collection %s", collection)
		}
	}

	if err != nil {
		return nil, err
	}
	if len(items) == 1 {
		it, err := loadItem(items, f, reqURL(r, r.URL.Path))
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
	collection := h.Typer.Type(r)
	repo := st.Loader{}

	id := chi.URLParam(r, "id")

	var items as.ItemCollection
	var err error
	f := &st.Filters{}
	f.FromRequest(r)
	f.ItemKey = []st.Hash{
		st.Hash(id),
	}
	f.MaxItems = 1

	if h.ValidObjectCollection(string(f.Collection)) {
		items, err = repo.LoadObjects(f)
	} else {
		// Non recognized as valid collection types
		// In our case activities
		switch f.Collection {
		case h.CollectionType("actors"):
			items, err = repo.LoadActors(f)
		case h.CollectionType("items"):
			items, err = repo.LoadObjects(f)
		default:
			return nil, errors.BadRequestf("invalid collection %s", f.Collection)
		}
	}
	if err != nil {
		return nil, err
	}
	if len(items) == 1 {
		it, err := loadItem(items, f, reqURL(r, r.URL.Path))
		if err != nil {
			return nil, errors.NotFoundf("%s", collection)
		}
		return it, nil
	}

	return nil, errors.NotFoundf("%s %s", collection, id)
}

func loadItem(items as.ItemCollection, f st.Paginator, baseURL string) (as.Item, error) {
	return items[0], nil
}
