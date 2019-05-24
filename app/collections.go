package app

import (
	"fmt"
	h "github.com/go-ap/activitypub/handler"
	"github.com/go-ap/activitypub/storage"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/fedbox/internal/context"
	"github.com/go-ap/fedbox/internal/errors"
	st "github.com/go-ap/fedbox/storage"
	"net/http"
)

func reqURL(r *http.Request, path string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s%s", scheme, r.Host, path)
}

// HandleActivityCollection serves content from the outbox, inbox, likes, shares and replies end-points
// that return ActivityPub collections containing activities
func HandleActivityCollection(w http.ResponseWriter, r *http.Request) (as.CollectionInterface, error) {
	var items as.ItemCollection
	var err error
	var total int
	f := &st.Filters{}
	f.FromRequest(r)

	var repo storage.ActivityLoader
	var ok bool
	if repo, ok = context.ActivityLoader(r.Context()); !ok {
		return nil, errors.Newf("invalid database connection")
	}
	items, total, err = repo.LoadActivities(f)

	if err != nil {
		return nil, err
	}
	it, err := loadCollection(items, uint(total), f, reqURL(r, r.URL.Path))
	if err != nil {
		return nil, NotFoundf("%s", f.Collection)
	}
	return it, nil
}

// HandleObjectCollection serves content from following, followers, liked, and likes end-points
// that return ActivityPub collections containing plain objects
func HandleObjectCollection(w http.ResponseWriter, r *http.Request) (as.CollectionInterface, error) {
	var items as.ItemCollection
	var err error
	var total int
	f := &st.Filters{}
	f.FromRequest(r)

	if h.ValidActivityCollection(string(f.Collection)) {
		var repo storage.ObjectLoader
		var ok bool
		if repo, ok = context.ObjectLoader(r.Context()); !ok {
			return nil, errors.Newf("invalid database connection")
		}
		items, total, err = repo.LoadObjects(f)
	} else {
		// Non recognized as valid collection types
		// In our case actors, items
		switch f.Collection {
		case ActorsType:
			var repo storage.ActorLoader
			var ok bool
			if repo, ok = context.ActorLoader(r.Context()); !ok {
				return nil, errors.Newf("invalid database connection")
			}
			items, total, err = repo.LoadActors(f)
		case ObjectsType:
			var repo storage.ObjectLoader
			var ok bool
			if repo, ok = context.ObjectLoader(r.Context()); !ok {
				return nil, errors.Newf("invalid database connection")
			}
			items, total, err = repo.LoadObjects(f)
		default:
			return nil, BadRequestf("invalid collection %s", f.Collection)
		}
	}
	if err != nil {
		return nil, err
	}
	it, err := loadCollection(items, uint(total), f, reqURL(r, r.URL.Path))
	if err != nil {
		return nil, NotFoundf("%s", f.Collection)
	}
	return it, nil
}


// HandleCollection serves content from the generic collection end-points
// that return ActivityPub objects or activities
func HandleCollection(w http.ResponseWriter, r *http.Request) (as.CollectionInterface, error) {
	var items as.CollectionInterface
	var err error

	f := &st.Filters{}
	f.FromRequest(r)
	f.ItemKey = append(f.ItemKey, st.Hash(reqURL(r, r.URL.Path)))

	var repo storage.CollectionLoader
	var ok bool
	if repo, ok = context.CollectionLoader(r.Context()); !ok {
		return nil, errors.Newf("invalid database connection")
	}
	items, _, err = repo.LoadCollection(f)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func loadCollection(items as.ItemCollection, count uint, filters st.Paginator, baseUrl string) (as.CollectionInterface, error) {
	getURL := func(f st.Paginator) string {
		qs := ""
		if f != nil {
			qs = f.QueryString()
		}
		return fmt.Sprintf("%s%s", baseUrl, qs)
	}

	var haveItems, moreItems, lessItems bool
	var bp, fp, cp, pp, np st.Paginator

	oc := as.OrderedCollection{}
	oc.ID = as.ObjectID(getURL(bp))
	oc.Type = as.OrderedCollectionType

	f, _ := filters.(*st.Filters)
	haveItems = len(items) > 0

	moreItems = int(count) > ((f.Page + 1) * f.MaxItems)
	lessItems = f.Page > 1
	if filters != nil {
		bp = filters.BasePage()
		fp = filters.FirstPage()
		cp = filters.CurrentPage()
	}

	if haveItems {
		oc.OrderedItems = items
		firstURL := getURL(fp)
		oc.First = as.IRI(firstURL)

		if f.Page >= 1 {
			curURL := getURL(cp)
			page := as.OrderedCollectionPageNew(&oc)
			page.ID = as.ObjectID(curURL)

			if moreItems {
				np = filters.NextPage()
				nextURL := getURL(np)
				page.Next = as.IRI(nextURL)
			}
			if lessItems {
				pp = filters.PrevPage()
				prevURL := getURL(pp)
				page.Prev = as.IRI(prevURL)
			}
			page.TotalItems = count
			return page, nil
		}
	}

	oc.TotalItems = count
	return &oc, nil
}
