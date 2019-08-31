package activitypub

import (
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/errors"
	"github.com/mariusor/qstring"
)

// Paginator
type Paginator interface {
	Count() uint
	Page() uint
}

func getURL(i as.IRI, f Paginator) as.IRI {
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
		i = as.IRI(u.String())
	}
	return i
}

// PaginateCollection is a function that populates the received collection as
func PaginateCollection(col as.CollectionInterface, f Paginator) (as.CollectionInterface, error) {
	if col == nil {
		return col, errors.Newf("unable to paginate nil collection")
	}

	baseURL := col.GetLink()
	curURL := getURL(baseURL, f)

	var haveItems, moreItems, lessItems bool
	var pp, np Paginator

	count := col.Count()
	haveItems = count > 0
	moreItems = count > ((f.Page() + 1) * f.Count())
	lessItems = f.Page() > 1

	if haveItems {
		var firstURL as.IRI

		maxItems := count
		if maxItems < MaxItems {
			maxItems = MaxItems
		}
		if f != nil {
			fp := &Filters{CurPage: 1, MaxItems: maxItems}
			firstURL = getURL(baseURL, fp)
		}
		if col.GetType() == as.OrderedCollectionType {
			oc, err := ToOrderedCollection(col)
			if err == nil && len(firstURL) > 0 {
				oc.First = firstURL
				col = oc
			}
		}
		if col.GetType() == as.CollectionType {
			c, err := ToCollection(col)
			if err == nil && len(firstURL) > 0 {
				c.First = firstURL
				col = c
			}
		}
		var nextURL, prevURL as.IRI
		if moreItems {
			np = &Filters{CurPage: f.Page() + 1, MaxItems: maxItems}
			nextURL = getURL(baseURL, np)
		}
		if lessItems {
			pp = &Filters{CurPage: f.Page() - 1, MaxItems: maxItems}
			prevURL = getURL(baseURL, pp)
		}

		if f.Page() > 0 {
			if col.GetType() == as.OrderedCollectionType {
				oc, err := ToOrderedCollection(col)
				if err == nil {
					page := as.OrderedCollectionPageNew(oc)
					page.ID = as.ObjectID(curURL)
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
					page.OrderedItems = oc.OrderedItems
					page.TotalItems = count
					col = page
				}
			}
			if col.GetType() == as.CollectionType {
				c, err := ToCollection(col)
				if err == nil {
					page := as.CollectionPageNew(c)
					page.ID = as.ObjectID(curURL)
					page.PartOf = baseURL
					page.First = c.First
					if moreItems {
						page.Next = nextURL
					}
					if lessItems {
						page.Prev = prevURL
					}
					page.TotalItems = count
					page.Items = c.Items
					col = page
				}
			}
			return col, nil
		}
	}

	return col, nil
}
