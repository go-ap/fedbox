package activitypub

import (
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	"github.com/mariusor/qstring"
	"math"
)

// Paginator
type Paginator interface {
	Count() uint
	Page() uint
}

func getURL(i pub.IRI, f Paginator) pub.IRI {
	q, err := qstring.Marshal(f)
	if err != nil {
		return i
	}
	if u, err := i.URL(); err == nil {
		query := u.Query()
		for k, el := range q {
			if len(el) == 0 {
				continue
			}
			for _, v := range el {
				query.Set(k, v)
			}
		}
		u.RawQuery = query.Encode()
		i = pub.IRI(u.String())
	}
	return i
}

func paginateItems(col pub.ItemCollection, f Paginator) (pub.ItemCollection, error) {
	if col == nil {
		return nil, nil
	}
	if f == nil {
		return nil, nil
	}
	if uint(len(col)) <= f.Count() {
		return col, nil
	}
	page := uint(math.Max(float64(f.Page()), 1.0))
	start := (page - 1) * f.Count()
	stop := uint(math.Min(float64(f.Count()), float64(uint(len(col))-start)))
	col = col[start : start+stop]

	return col, nil
}

// PaginateCollection is a function that populates the received collection pub
func PaginateCollection(col pub.CollectionInterface, f Paginator) (pub.CollectionInterface, error) {
	if col == nil {
		return col, errors.Newf("unable to paginate nil collection")
	}

	u, _ := col.GetLink().URL()
	u.RawQuery = ""
	baseURL := pub.IRI(u.String())
	curURL := getURL(baseURL, f)

	var haveItems, moreItems, lessItems bool
	var pp, np Paginator

	count := col.Count()
	maxItems := f.Count()
	haveItems = count > 0
	moreItems = count > ((f.Page()) * maxItems)
	lessItems = f.Page() > 1

	if haveItems {
		var firstURL pub.IRI

		if f != nil {
			fp := &Filters{CurPage: 1, MaxItems: maxItems}
			firstURL = getURL(baseURL, fp)
		}
		if col.GetType() == pub.OrderedCollectionType {
			oc, err := pub.ToOrderedCollection(col)
			if err == nil && len(firstURL) > 0 {
				oc.First = firstURL
				oc.OrderedItems, _ = paginateItems(oc.OrderedItems, f)
				col = oc
			}
		}
		if col.GetType() == pub.CollectionType {
			c, err := pub.ToCollection(col)
			if err == nil && len(firstURL) > 0 {
				c.First = firstURL
				c.Items, _ = paginateItems(c.Items, f)
				col = c
			}
		}
		var nextURL, prevURL pub.IRI
		if moreItems {
			np = &Filters{CurPage: f.Page() + 1, MaxItems: maxItems}
			nextURL = getURL(baseURL, np)
		}
		if lessItems {
			pp = &Filters{CurPage: f.Page() - 1, MaxItems: maxItems}
			prevURL = getURL(baseURL, pp)
		}

		if f.Page() > 0 {
			if col.GetType() == pub.OrderedCollectionType {
				oc, err := pub.ToOrderedCollection(col)
				if err == nil {
					page := pub.OrderedCollectionPageNew(oc)
					page.ID = pub.ID(curURL)
					page.PartOf = baseURL
					if firstURL != curURL {
						page.First = oc.First
					}
					if moreItems {
						page.Next = nextURL
					}
					if lessItems {
						page.Prev = prevURL
					}
					page.OrderedItems, _ = paginateItems(oc.OrderedItems, f)
					page.TotalItems = count
					col = page
				}
			}
			if col.GetType() == pub.CollectionType {
				c, err := pub.ToCollection(col)
				if err == nil {
					page := pub.CollectionPageNew(c)
					page.ID = pub.ID(curURL)
					page.PartOf = baseURL
					page.First = c.First
					if moreItems {
						page.Next = nextURL
					}
					if lessItems {
						page.Prev = prevURL
					}
					page.TotalItems = count
					page.Items, _ = paginateItems(c.Items, f)
					col = page
				}
			}
			return col, nil
		}
	}

	return col, nil
}
