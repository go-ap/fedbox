package app

import (
	h "github.com/go-ap/activitypub/handler"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/errors"
	"github.com/go-ap/fedbox/storage"
	"github.com/go-chi/chi"
	"net/http"
)

// HandleItem serves content from the following, followers, liked, and likes end-points
// that returns a single ActivityPub object
func HandleItem(w http.ResponseWriter, r *http.Request) (as.Item, error) {
	collection := h.Typer.Type(r)

	id := chi.URLParam(r, "id")

	var items as.ItemCollection
	var err error
	ff, err := activitypub.FromRequest(r)
	f, _ := ff.(*activitypub.Filters)
	LoadToFilters(r, f)

	if len(f.ItemKey) == 0 {
		f.ItemKey = []activitypub.Hash{
			activitypub.Hash(id),
		}
	}
	f.MaxItems = 1

	if !activitypub.ValidActivityCollection(string(f.Collection)) {
		return nil, NotFoundf("collection '%s' not found", f.Collection)
	}
	var repo storage.Loader
	if h.ValidObjectCollection(string(f.Collection)) {
		var ok bool
		if repo, ok = Loader(r.Context()); !ok {
			return nil, errors.Newf("invalid object loader")
		}
		items, _, err = repo.LoadObjects(f)
	} else if activitypub.ValidActivityCollection(string(f.Collection)) {
		var ok bool
		if repo, ok = Loader(r.Context()); !ok {
			return nil, errors.Newf("invalid activity loader")
		}
		items, _, err = repo.LoadActivities(f)
	}

	switch f.Collection {
	case activitypub.ActivitiesType:
		var ok bool
		if repo, ok = Loader(r.Context()); !ok {
			return nil, errors.Newf("invalid activity loader")
		}
		items, _, err = repo.LoadActivities(f)
	case activitypub.ActorsType:
		var ok bool
		if repo, ok = Loader(r.Context()); !ok {
			return nil, errors.Newf("invalid database connection")
		}
		items, _, err = repo.LoadActors(f)
	case activitypub.ObjectsType:
		var ok bool
		if repo, ok = Loader(r.Context()); !ok {
			return nil, errors.Newf("invalid database connection")
		}
		items, _, err = repo.LoadObjects(f)
	default:
		return nil, errors.Newf("invalid collection %s", f.Collection)
	}
	if err != nil {
		return nil, err
	}
	if len(items) == 1 {
		it, err := loadItem(items, f, reqURL(r))
		if err != nil {
			return nil, NotFoundf("Not found %s", collection)
		}
		return it, nil
	}

	return nil, NotFoundf("Not found %s in %s", id, collection)
}

func loadItem(items as.ItemCollection, f activitypub.Paginator, baseURL string) (as.Item, error) {
	return items[0], nil
}
