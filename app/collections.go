package app

import (
	"fmt"
	h "github.com/go-ap/activitypub/handler"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/fedbox/internal/errors"
	j "github.com/go-ap/jsonld"
	"github.com/go-chi/chi"
	"net/http"
)

func renderCollection(c as.CollectionInterface) ([]byte, error) {
	return j.WithContext(j.IRI(as.ActivityBaseURI)).Marshal(c)
}

// HandleActivityCollection serves content from the outbox, inbox, likes, shares and replies end-points
// that return ActivityPub collections containing activities
func HandleActivityCollection(w http.ResponseWriter, r *http.Request) (as.CollectionInterface, error) {
	collection := h.Typer.Type(r)
	repo := loader{}

	var items as.ItemCollection
	var err error
	f := Filters{}
	f.FromRequest(r)

	if col := chi.URLParam(r, "collection"); len(col) > 0 {
		if h.CollectionType(col) == collection {
			if h.ValidActivityCollection(col) {
				items, err = repo.LoadActivities(f)
			} else {
				return nil, errors.BadRequestf("invalid collection %s", collection)
			}
		}
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
	if len(items) > 0 {
		it, err := loadCollection(items, uint(len(items)), &f, reqURL(r, r.URL.Path))
		if err != nil {
			return nil, errors.NotFoundf("%s", collection)
		}
		return it, nil
	}
	return nil, errors.NotImplementedf("%s", collection)
}

// HandleObjectCollection serves content from following, followers, liked, and likes end-points
// that return ActivityPub collections containing plain objects
func HandleObjectCollection(w http.ResponseWriter, r *http.Request) (as.CollectionInterface, error) {
	collection := h.Typer.Type(r)
	repo := loader{}
	var items as.ItemCollection
	var err error
	f := Filters{}
	f.FromRequest(r)

	if col := chi.URLParam(r, "collection"); len(col) > 0 {
		if h.CollectionType(col) == collection {
			if h.ValidObjectCollection(col) {
				items, err = repo.LoadObjects(f)
			} else {
				return nil, errors.BadRequestf("invalid collection %s", collection)
			}
		}
	} else {
		// Non recognized as valid collection types
		// In our case actors and items
		switch collection {
		case h.CollectionType("actors"):
			items, err = repo.LoadActors(f)
		case h.CollectionType("items"):
			items, err = repo.LoadObjects(f)
		default:
			return nil, errors.BadRequestf("invalid collection %s", collection)
		}
	}
	if err != nil {
		return nil, err
	}
	if len(items) > 0 {
		it, err := loadCollection(items, uint(len(items)), &f, reqURL(r, r.URL.Path))
		if err != nil {
			return nil, errors.NotFoundf("%s", collection)
		}
		return it, nil
	}
	return nil, errors.NotImplementedf("%s", collection)
}

func loadCollection(items as.ItemCollection, count uint, filters Paginator, baseUrl string) (as.CollectionInterface, error) {
	getURL := func(f Paginator) string {
		qs := ""
		if f != nil {
			qs = f.QueryString()
		}
		return fmt.Sprintf("%s%s", baseUrl, qs)
	}

	var haveItems, moreItems, lessItems bool
	var bp, fp, cp, pp, np Paginator

	oc := as.OrderedCollection{}
	oc.ID = as.ObjectID(getURL(bp))
	oc.Type = as.OrderedCollectionType

	f, _ := filters.(*Filters)
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
