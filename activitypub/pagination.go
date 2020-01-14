package activitypub

import (
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	"github.com/mariusor/qstring"
	"math"
	"path"
)

// KeysetPaginator
type KeysetPaginator interface {
	Before() Hash
	After() Hash
	Count() uint
}

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
	count := f.Count()
	if count == 0 {
		count = MaxItems
	}

	if uint(len(col)) <= f.Count() {
		return col, nil
	}
	var start, stop uint
	if ff, ok := f.(KeysetPaginator); ok {
		stop = uint(math.Min(float64(count), float64(len(col))))
		if len(ff.Before())+len(ff.After()) > 0 {
			for i, it := range col {
				if len(ff.Before()) > 0 {
					if ff.Before().Matches(it.GetLink()) {
						start = uint(math.Max(0, float64(i - int(count))))
					}
				}
				if len(ff.After()) > 0 {
					if ff.After().Matches(it.GetLink()) {
						start = uint(i + 1)
					}
				}
			}
		}
	} else {f.Count()
		page := uint(math.Max(float64(f.Page()), 1.0))
		start = (page - 1) * f.Count()
		if start > col.Count() {
			start = 0
		}
	}
	stop = uint(math.Min(float64(count), float64(uint(len(col))-start)))
	if stop == 0 {
		stop = uint(len(col))
	}
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
	if maxItems == 0 {
		maxItems = MaxItems
	}
	haveItems = count > 0
	if _, ok := f.(KeysetPaginator); ok {
		moreItems = count > maxItems
		lessItems = true
	} else {
		moreItems = count > ((f.Page()) * maxItems)
		lessItems = f.Page() > 1
	}

	ordered := pub.ActivityVocabularyTypes{
		pub.OrderedCollectionPageType,
		pub.OrderedCollectionType,
	}
	unOrdered := pub.ActivityVocabularyTypes{
		pub.CollectionPageType,
		pub.CollectionType,
	}

	// TODO(marius): refactor this with OnCollection functions
	if haveItems {
		var firstURL pub.IRI

		if f != nil {
			fp := &Filters{MaxItems: maxItems}
			if _, ok := f.(KeysetPaginator); !ok {
				fp.CurPage = 1
			}
			firstURL = getURL(baseURL, fp)
		}
		if ordered.Contains(col.GetType()) {
			pub.OnOrderedCollection(col, func(oc *pub.OrderedCollection) error {
				if len(firstURL) > 0 {
					oc.First = firstURL
					oc.OrderedItems, _ = paginateItems(oc.OrderedItems, f)
				}
				return nil
			})
		}
		if unOrdered.Contains(col.GetType()) {
			pub.OnCollection(col, func(c *pub.Collection) error {
				c.First = firstURL
				c.Items, _ = paginateItems(c.Items, f)
				return nil
			})
		}
		var nextURL, prevURL pub.IRI
		if moreItems {
			if _, ok := f.(KeysetPaginator); ok {
				items := col.Collection()
				uuid := path.Base(items[len(items)-1].GetLink().String())
				np = &Filters{First: Hash(uuid), MaxItems: maxItems}
			} else {
				np = &Filters{CurPage: f.Page() + 1, MaxItems: maxItems}
			}
			nextURL = getURL(baseURL, np)
		}
		if lessItems {
			if _, ok := f.(KeysetPaginator); ok {
				items := col.Collection()
				uuid := path.Base(items.First().GetLink().String())
				pp = &Filters{Last: Hash(uuid), MaxItems: maxItems}
			} else {
				pp = &Filters{CurPage: f.Page() - 1, MaxItems: maxItems}
			}
			prevURL = getURL(baseURL, pp)
		}

		if f.Count() > 0 {
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
