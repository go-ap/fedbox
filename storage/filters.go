package storage

import (
	"fmt"
	h "github.com/go-ap/activitypub/handler"
	s "github.com/go-ap/activitypub/storage"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/fedbox/internal/errors"
	"github.com/mariusor/qstring"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

// Hash
type Hash string

// String returns the hash as a string
func (h Hash) String() string {
	return string(h)
}

// Filters
type Filters struct {
	Collection   h.CollectionType            `qstring:"-"`
	Key          []Hash                      `qstring:"id,omitempty"`
	ItemKey      []Hash                      `qstring:"objectid,omitempty"`
	Type         []as.ActivityVocabularyType `qstring:"type"`
	AttributedTo []Hash                      `qstring:"attributedTo,omitempty"`
	FollowedBy   []Hash                      `qstring:"followedBy,omitempty"` // todo(marius): not really used
	OlderThan    time.Time                   `qstring:"olderThan,omitempty"`
	NewerThan    time.Time                   `qstring:"newerThan,omitempty"`
	Page         int                         `qstring:"page,omitempty"`
	MaxItems     int                         `qstring:"maxItems,omitempty"`
}

// IRIs returns a list of ActivityVocabularyTypes to filter against
func (f Filters) Types() []as.ActivityVocabularyType {
	return f.Type
}

// IRIs returns a list of IRIs to filter against
func (f Filters) IRIs() []as.IRI {
	ret := make([]as.IRI, len(f.ItemKey))
	for i, k := range f.ItemKey {
		ret[i] = as.IRI(k)
	}
	return ret
}
// FromRequest loads the filters we use for generating storage queries from the HTTP request
func (f *Filters) FromRequest(r *http.Request) error {
	if err := qstring.Unmarshal(r.URL.Query(), f); err != nil {
		return err
	}

	f.Collection = h.Typer.Type(r)

	pr, _ := regexp.Compile(fmt.Sprintf("/(actors|items|activities)/(\\w+)/%s", f.Collection))
	matches := pr.FindSubmatch([]byte(r.URL.Path))
	if len(matches) < 3 {
		errors.NotFoundf("%s", "actor")
	} else {
		col := matches[1]
		switch string(col) {
		case "actors":
			f.AttributedTo = []Hash{
				Hash(matches[2]),
			}
		case "items":
			f.ItemKey = []Hash{
				Hash(matches[2]),
			}
		case "activities":
			f.Key = []Hash{
				Hash(matches[2]),
			}
		}
	}
	return nil
}

func copyActivityFilters(dst *Filters, src Filters) {
	dst.Type = src.Type
	dst.Key = src.Key
	dst.ItemKey = src.ItemKey
	dst.Type = src.Type
	dst.AttributedTo = src.AttributedTo
	dst.FollowedBy = src.FollowedBy
	dst.OlderThan = src.OlderThan
	dst.NewerThan = src.NewerThan
	dst.Page = src.Page
	dst.MaxItems = src.MaxItems
}

func query(f *Filters) string {
	res := ""

	var u url.Values
	var err error
	if u, err = qstring.Marshal(f); err != nil {
		return ""
	}

	if len(u) > 0 {
		res = "?" + u.Encode()
	}
	return res
}

// QueryString
func (a *Filters) QueryString() string {
	return query(a)
}

// BasePage
func (a *Filters) BasePage() s.Paginator {
	b := &Filters{}
	copyActivityFilters(b, *a)
	return b
}

// CurrentPage
func (a *Filters) CurrentPage() s.Paginator {
	return a
}

// NextPage
func (a *Filters) NextPage() s.Paginator {
	b := &Filters{}
	copyActivityFilters(b, *a)
	b.Page += 1
	return b
}

// PrevPage
func (a *Filters) PrevPage() s.Paginator {
	b := &Filters{}
	copyActivityFilters(b, *a)
	b.Page -= 1
	return b
}

// FirstPage
func (a *Filters) FirstPage() s.Paginator {
	b := &Filters{}
	copyActivityFilters(b, *a)
	b.Page = 1
	return b
}

// CurrentIndex
func (a *Filters) CurrentIndex() int {
	return a.Page
}

//type Filterable interface {
//	GetWhereClauses() ([]string, []interface{})
//	GetLimit() string
//}
//func (a Filters) GetWhereClauses() ([]string, []interface{}) {
//	return []string{}, []interface{}{}
//}
//func (a Filters) GetLimit() string {
//	if a.MaxItems == 0 {
//		return ""
//	}
//	limit := fmt.Sprintf("  LIMIT %d", a.MaxItems)
//	if a.Page > 1 {
//		limit = fmt.Sprintf("%s OFFSET %d", limit, a.MaxItems*(a.Page-1))
//	}
//	return limit
//}
