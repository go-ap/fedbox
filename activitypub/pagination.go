package activitypub

import (
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	"github.com/mariusor/qstring"
	"math"
	"path"
	"time"
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
		u.RawQuery = q.Encode()
		i = pub.IRI(u.String())
	}
	return i
}

func paginateItems(col pub.ItemCollection, f Paginator) (pub.ItemCollection, string, string, error) {
	var prev, next string
	if pub.IsNil(col) {
		return nil, prev, next, nil
	}
	if f == nil {
		return nil, prev, next, nil
	}
	count := f.Count()
	if count == 0 {
		count = MaxItems
	}

	if uint(len(col)) <= f.Count() {
		return col, prev, next, nil
	}
	var start, stop int
	if ff, ok := f.(KeysetPaginator); ok {
		stop = int(math.Min(float64(count), float64(len(col))))
		if len(ff.Before())+len(ff.After()) > 0 {
			for i, it := range col {
				if len(ff.Before()) > 0 {
					if ff.Before().Matches(it.GetLink()) {
						start = int(math.Max(0, float64(i-int(count))))
					}
				}
				if len(ff.After()) > 0 {
					if ff.After().Matches(it.GetLink()) {
						start = int(i + 1)
					}
				}
			}
		}
	} else {
		f.Count()
		page := int(math.Max(float64(f.Page()), 1.0))
		start = (page - 1) * int(f.Count())
		if start > int(col.Count()) {
			start = 0
		}
	}
	stop = int(math.Min(float64(count), float64(len(col)-start)))
	if stop == 0 {
		stop = len(col)
	} else {
		stop = start + stop
	}
	cnt := len(col)
	if start > 0 && start < cnt {
		prev = path.Base(col[start].GetLink().String())
	}
	if stop < cnt {
		next = path.Base(col[stop-1].GetLink().String())
	}
	col = col[start:stop]
	return col, prev, next, nil
}

func copyFilter(f *Filters, p Paginator) {
	if ff, ok := p.(*Filters); ok {
		f.baseURL = ff.baseURL
		f.Name = ff.Name
		f.Cont = ff.Cont
		f.Authenticated = ff.Authenticated
		f.To = ff.To
		f.Author = ff.Author
		f.Parent = ff.Parent
		f.IRI = ff.IRI
		f.Collection = ff.Collection
		f.URL = ff.URL
		f.MedTypes = ff.MedTypes
		f.Aud = ff.Aud
		f.Key = ff.Key
		f.ItemKey = ff.ItemKey
		if ff.Object != nil {
			if f.Object == nil {
				f.Object = FiltersNew()
			}
			copyFilter(f.Object, ff.Object)
		}
		if ff.Actor != nil {
			if f.Actor == nil {
				f.Actor = FiltersNew()
			}
			copyFilter(f.Actor, ff.Actor)
		}
		if ff.Target != nil {
			if f.Target == nil {
				f.Target = FiltersNew()
			}
			copyFilter(f.Target, ff.Target)
		}
		f.Type = ff.Type
		f.AttrTo = ff.AttrTo
		f.InReplTo = ff.InReplTo
		f.OP = ff.OP
		f.FollowedBy = ff.FollowedBy
		f.OlderThan = ff.OlderThan
		f.NewerThan = ff.NewerThan
	}
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

	var haveItems bool
	var prev, next string // uuids

	count := col.Count()
	maxItems := f.Count()
	if maxItems == 0 {
		maxItems = MaxItems
	}
	haveItems = count > 0

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
			fp := FiltersNew()
			copyFilter(fp, f)
			fp.MaxItems = maxItems
			if _, ok := f.(KeysetPaginator); !ok {
				fp.CurPage = 1
			}
			firstURL = getURL(baseURL, fp)
		}
		if col.GetType() == pub.CollectionOfItems {
			err := pub.OnItemCollection(col, func(items *pub.ItemCollection) error {
				*items, _, _, _ = paginateItems(items.Collection(), f)
				return nil
			})
			return col, err
		}
		if ordered.Contains(col.GetType()) {
			pub.OnOrderedCollection(col, func(oc *pub.OrderedCollection) error {
				if len(firstURL) > 0 {
					oc.First = firstURL
				}
				oc.OrderedItems, prev, next, _ = paginateItems(oc.OrderedItems, f)
				return nil
			})
		}
		if unOrdered.Contains(col.GetType()) {
			pub.OnCollection(col, func(c *pub.Collection) error {
				c.First = firstURL
				c.Items, prev, next, _ = paginateItems(c.Items, f)
				return nil
			})
		}
		var nextURL, prevURL pub.IRI
		if len(next) > 0 {
			np := FiltersNew()
			copyFilter(np, f)
			np.MaxItems = maxItems
			if _, ok := f.(KeysetPaginator); ok {
				np.Next = Hash(next)
			} else {
				np.CurPage = f.Page() + 1
			}
			nextURL = getURL(baseURL, np)
		}
		if len(prev) > 0 {
			pp := FiltersNew()
			copyFilter(pp, f)
			pp.MaxItems = maxItems
			if _, ok := f.(KeysetPaginator); ok {
				pp.Prev = Hash(prev)
			} else {
				pp.CurPage = f.Page() + 1
			}
			prevURL = getURL(baseURL, pp)
		}

		if f.Count() > 0 {
			if col.GetType() == pub.OrderedCollectionType {
				oc, err := pub.ToOrderedCollection(col)
				if err == nil {
					page := pub.OrderedCollectionPageNew(oc)
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
					page.TotalItems = count
					col = page
				}
			}
			if col.GetType() == pub.CollectionType {
				c, err := pub.ToCollection(col)
				if err == nil {
					page := pub.CollectionPageNew(c)
					page.ID = curURL
					page.PartOf = baseURL
					page.First = c.First
					if len(nextURL) > 0 {
						page.Next = nextURL
					}
					if len(prevURL) > 0 {
						page.Prev = prevURL
					}
					page.TotalItems = count
					page.Items, _, _, _ = paginateItems(c.Items, f)
					col = page
				}
			}
		}
	}
	updatedAt := time.Time{}
	for _, it := range col.Collection() {
		pub.OnObject(it, func(o *pub.Object) error {
			if o.Published.Sub(updatedAt) > 0 {
				updatedAt = o.Published
			}
			if o.Updated.Sub(updatedAt) > 0 {
				updatedAt = o.Updated
			}
			return nil
		})
	}
	pub.OnObject(col, func(o *pub.Object) error {
		o.Updated = updatedAt
		return nil
	})

	return col, nil
}
