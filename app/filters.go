package app

import (
	"fmt"
	as "github.com/go-ap/activitystreams"
	"github.com/mariusor/qstring"
	"net/http"
	"net/url"
	"time"
)

type Filterable interface {
	Types() []as.ActivityVocabularyType
	IRIs() []as.IRI
}

// Paginator
type Paginator interface {
	QueryString() string
	BasePage() Paginator
	CurrentPage() Paginator
	NextPage() Paginator
	PrevPage() Paginator
	FirstPage() Paginator
	CurrentIndex() int
}

// Hash
type Hash string

// String returns the hash as a string
func (h Hash) String() string {
	return string(h)
}

// Filters
type Filters struct {
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
func (f Filters) Types() []as.ActivityVocabularyType {
	return f.Type
}
func (f Filters) IRIs() []as.IRI {
	ret := make([]as.IRI, len(f.ItemKey))
	for i, k := range f.ItemKey {
		ret[i] = as.IRI(k)
	}
 	return ret
}
func (f *Filters) FromRequest(r *http.Request) error {
	return qstring.Unmarshal(r.URL.Query(), f)
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

func reqURL(r *http.Request, path string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s%s", scheme, r.Host, path)
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
func (a *Filters) BasePage() Paginator {
	b := &Filters{}
	copyActivityFilters(b, *a)
	return b
}

// CurrentPage
func (a *Filters) CurrentPage() Paginator {
	return a
}

// NextPage
func (a *Filters) NextPage() Paginator {
	b := &Filters{}
	copyActivityFilters(b, *a)
	b.Page += 1
	return b
}

// PrevPage
func (a *Filters) PrevPage() Paginator {
	b := &Filters{}
	copyActivityFilters(b, *a)
	b.Page -= 1
	return b
}

// FirstPage
func (a *Filters) FirstPage() Paginator {
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
