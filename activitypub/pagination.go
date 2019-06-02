package activitypub

import (
	"fmt"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/errors"
)

type Paginator interface {
	QueryString() string
	FirstPage() Paginator
	CurrentPage() Paginator
	NextPage() Paginator
	PrevPage() Paginator
}

func GetPaginatedCollection(col as.CollectionInterface, filters Paginator) (as.CollectionInterface, error) {
	if col == nil {
		return col, errors.Newf("unable to paginate nil collection")
	}

	var count uint
	baseURL := col.GetLink()
	getURL := func(f Paginator) string {
		qs := ""
		if f != nil {
			qs = f.QueryString()
		}
		return fmt.Sprintf("%s%s", baseURL, qs)
	}

	var items as.ItemCollection
	var haveItems, moreItems, lessItems bool
	var fp, cp, pp, np Paginator

	if col.GetType() == as.OrderedCollectionType {
		oc, err := ToOrderedCollection(col)
		if err == nil {
			count = oc.TotalItems
			col = oc
			items = oc.OrderedItems
		}
	}
	if col.GetType() == as.CollectionType {
		c, err := ToCollection(col)
		if err == nil {
			count = c.TotalItems
			col = c
			items = c.Items
		}
	}

	f, _ := filters.(*Filters)
	haveItems = len(items) > 0

	moreItems = int(count) > ((f.Page + 1) * f.MaxItems)
	lessItems = f.Page > 1
	if filters != nil {
		fp = filters.FirstPage()
		cp = filters.CurrentPage()
	}

	if haveItems {
		firstURL := getURL(fp)
		if col.GetType() == as.OrderedCollectionType {
			oc, err := ToOrderedCollection(col)
			if err == nil {
				oc.First = as.IRI(firstURL)
				col = oc
			}
		}
		if col.GetType() == as.CollectionType {
			c, err := ToCollection(col)
			if err == nil {
				c.First = as.IRI(firstURL)
				col = c
			}
		}

		if f.Page >= 1 {
			var nextURL string
			var prevURL string
			curURL := getURL(cp)
			if moreItems {
				np = filters.NextPage()
				nextURL = getURL(np)
			}
			if lessItems {
				pp = filters.PrevPage()
				prevURL = getURL(pp)
			}

			if col.GetType() == as.OrderedCollectionType {
				oc, err := ToOrderedCollection(col)
				if err == nil {
					page := as.OrderedCollectionPageNew(oc)
					page.ID = as.ObjectID(curURL)
					page.PartOf = baseURL
					page.First = oc.First
					if moreItems {
						page.Next = as.IRI(nextURL)
					}
					if lessItems {
						page.Prev = as.IRI(prevURL)
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
						page.Next = as.IRI(nextURL)
					}
					if lessItems {
						page.Prev = as.IRI(prevURL)
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
