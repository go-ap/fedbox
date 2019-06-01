package app

import (
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/activitypub"
	st "github.com/go-ap/fedbox/storage"
	h "github.com/go-ap/handlers"
	"github.com/go-ap/storage"
	"github.com/go-chi/chi"
	"net/http"
)

// HandleItem serves content from the following, followers, liked, and likes end-points
// that returns a single ActivityPub object
func HandleItem(r *http.Request, repo storage.ObjectLoader) (as.Item, error) {
	collection := h.Typer.Type(r)

	id := chi.URLParam(r, "id")

	var items as.ItemCollection
	var err error
	ff, err := activitypub.FromRequest(r)
	f, _ := ff.(*activitypub.Filters)
	LoadToFilters(r, f)

	if len(f.ItemKey) == 0 {
		f.ItemKey = []activitypub.Hash{
			activitypub.Hash(reqURL(r)),
		}
	}
	f.MaxItems = 1

	if !activitypub.ValidActivityCollection(string(f.Collection)) {
		return nil, errors.NotFoundf("collection '%s' not found", f.Collection)
	}
	if h.ValidObjectCollection(string(f.Collection)) {
		if obLoader, ok := repo.(storage.ObjectLoader); ok {
			items, _, err = obLoader.LoadObjects(f)
		}
	} else if h.ValidActivityCollection(string(f.Collection)) {
		if actLoader, ok := repo.(storage.ActivityLoader); ok {
			items, _, err = actLoader.LoadActivities(f)
		}
	} else {
		switch f.Collection {
		case activitypub.ActivitiesType:
			if actLoader, ok := repo.(storage.ActivityLoader); ok {
				items, _, err = actLoader.LoadActivities(f)
			}
		case activitypub.ActorsType:
			if actLoader, ok := repo.(st.ActorLoader); ok {
				items, _, err = actLoader.LoadActors(f)
			}
		case activitypub.ObjectsType:
			if obLoader, ok := repo.(storage.ObjectLoader); ok {
				items, _, err = obLoader.LoadObjects(f)
			}
		default:
			return nil, errors.Newf("invalid collection %s", f.Collection)
		}
	}
	if err != nil {
		return nil, err
	}
	if len(items) == 1 {
		it, err := loadItem(items, f, reqURL(r))
		if err != nil {
			return nil, errors.NotFoundf("Not found %s", collection)
		}
		return it, nil
	}

	return nil, errors.NotFoundf("Not found %s in %s", id, collection)
}

func loadItem(items as.ItemCollection, f activitypub.Paginator, baseURL string) (as.Item, error) {
	return items[0], nil
}
