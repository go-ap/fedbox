package activitypub

import (
	"fmt"
	"math"
	"net/url"
	"path"
	"time"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	"github.com/go-ap/filters"
	"github.com/mariusor/qstring"
)

// KeysetPaginator
type KeysetPaginator interface {
	Before() string
	After() string
	Count() int
}

// Paginator
type Paginator interface {
	Count() int
	Page() int
}

func getURL(i vocab.IRI, f Paginator) vocab.IRI {
	q, err := qstring.Marshal(f)
	if err != nil {
		return i
	}
	if u, err := i.URL(); err == nil {
		u.RawQuery = q.Encode()
		i = vocab.IRI(u.String())
	}
	return i
}

const MaxItems = 100

func paginateItems(col vocab.ItemCollection, f Paginator) (vocab.ItemCollection, string, string, error) {
	var prev, next string
	if vocab.IsNil(col) {
		return nil, prev, next, nil
	}
	if f == nil {
		return nil, prev, next, nil
	}
	count := f.Count()
	if count <= 0 {
		count = MaxItems
	}

	if len(col) <= f.Count() {
		return col, prev, next, nil
	}
	var start, stop int
	if ff, ok := f.(KeysetPaginator); ok {
		stop = int(math.Min(float64(count), float64(len(col))))
		if len(ff.Before())+len(ff.After()) > 0 {
			for i, it := range col {
				if len(ff.Before()) > 0 {
					if it.GetLink().Contains(vocab.IRI(ff.Before()), true) {
						start = int(math.Max(0, float64(i-count)))
					}
				}
				if len(ff.After()) > 0 {
					if it.GetLink().Contains(vocab.IRI(ff.After()), true) {
						start = i + 1
					}
				}
			}
		}
	} else {
		f.Count()
		page := int(math.Max(float64(f.Page()), 1.0))
		start = (page - 1) * f.Count()
		if start > int(col.Count()) {
			start = 0
		}
	}
	stop = int(math.Min(float64(count), float64(len(col)-start)))
	if stop == 0 || start+stop > 0 {
		stop = len(col)
	} else {
		stop = start + stop
	}
	cnt := len(col)
	if start > 0 && start < cnt {
		prev = path.Base(col[start].GetLink().String())
	}
	if stop > 0 && stop < cnt {
		next = path.Base(col[stop-1].GetLink().String())
	}
	col = col[start:stop]
	return col, prev, next, nil
}

// PaginateCollection is a function that populates the received collection
func PaginateCollection(col vocab.CollectionInterface, f Paginator) (vocab.CollectionInterface, error) {
	if col == nil {
		return col, errors.Newf("unable to paginate nil collection")
	}

	u, _ := col.GetLink().URL()
	u.User = nil
	u.RawQuery = ""
	baseURL := vocab.IRI(u.String())
	curURL := getURL(baseURL, f)

	var haveItems bool
	var prev, next string // uuids

	count := col.Count()
	maxItems := f.Count()
	if maxItems <= 0 {
		maxItems = MaxItems
	}
	haveItems = count > 0

	ordered := vocab.ActivityVocabularyTypes{
		vocab.OrderedCollectionPageType,
		vocab.OrderedCollectionType,
	}
	unOrdered := vocab.ActivityVocabularyTypes{
		vocab.CollectionPageType,
		vocab.CollectionType,
	}

	// TODO(marius): refactor this with OnCollection functions
	if haveItems {
		var firstURL vocab.IRI

		if f != nil {
			fp := url.Values{}
			fp.Set("maxItems", fmt.Sprintf("%d", maxItems))
			if _, ok := f.(KeysetPaginator); !ok {
				fp.Set("page", fmt.Sprintf("%d", 1))
			}
			firstURL = getURL(baseURL, filters.Paginator(fp))
		}
		if col.GetType() == vocab.CollectionOfItems {
			err := vocab.OnItemCollection(col, func(items *vocab.ItemCollection) error {
				*items, _, _, _ = paginateItems(items.Collection(), f)
				return nil
			})
			return col, err
		}
		if ordered.Contains(col.GetType()) {
			vocab.OnOrderedCollection(col, func(oc *vocab.OrderedCollection) error {
				if len(firstURL) > 0 {
					oc.First = firstURL
				}
				oc.OrderedItems, prev, next, _ = paginateItems(oc.OrderedItems, f)
				return nil
			})
		}
		if unOrdered.Contains(col.GetType()) {
			vocab.OnCollection(col, func(c *vocab.Collection) error {
				c.First = firstURL
				c.Items, prev, next, _ = paginateItems(c.Items, f)
				return nil
			})
		}
		var nextURL, prevURL vocab.IRI
		if len(next) > 0 {
			np := url.Values{}
			np.Set("maxItems", fmt.Sprintf("%d", maxItems))
			if _, ok := f.(KeysetPaginator); ok {
				np.Set("after", next)
			} else {
				np.Set("page", fmt.Sprintf("%d", f.Page()+1))
			}
			nextURL = getURL(baseURL, filters.Paginator(np))
		}
		if len(prev) > 0 {
			pp := url.Values{}
			pp.Set("maxItems", fmt.Sprintf("%d", maxItems))
			if _, ok := f.(KeysetPaginator); ok {
				pp.Set("before", prev)
			} else {
				pp.Set("page", fmt.Sprintf("%d", f.Page()+1))
			}
			prevURL = getURL(baseURL, filters.Paginator(pp))
		}

		if f.Count() > 0 {
			if col.GetType() == vocab.OrderedCollectionType {
				oc, err := vocab.ToOrderedCollection(col)
				if err == nil {
					page := vocab.OrderedCollectionPageNew(oc)
					page.ID = curURL
					page.PartOf = baseURL
					if firstURL != curURL {
						page.First = oc.First
					}
					if len(nextURL) > 0 {
						page.Next = nextURL
					}
					if len(prevURL) > 0 {
						page.Prev = prevURL
					}
					page.OrderedItems, _, _, _ = paginateItems(oc.OrderedItems, f)
					col = page
				}
			}
			if col.GetType() == vocab.CollectionType {
				c, err := vocab.ToCollection(col)
				if err == nil {
					page := vocab.CollectionPageNew(c)
					page.ID = curURL
					page.PartOf = baseURL
					page.First = c.First
					if len(nextURL) > 0 {
						page.Next = nextURL
					}
					if len(prevURL) > 0 {
						page.Prev = prevURL
					}
					page.Items, _, _, _ = paginateItems(c.Items, f)
					col = page
				}
			}
		}
	}
	updatedAt := time.Time{}
	for _, it := range col.Collection() {
		vocab.OnObject(it, func(o *vocab.Object) error {
			if o.Published.Sub(updatedAt) > 0 {
				updatedAt = o.Published
			}
			if o.Updated.Sub(updatedAt) > 0 {
				updatedAt = o.Updated
			}
			return nil
		})
	}
	vocab.OnObject(col, func(o *vocab.Object) error {
		o.Updated = updatedAt
		return nil
	})

	return col, nil
}
