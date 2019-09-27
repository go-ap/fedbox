package activitypub

import (
	"fmt"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	h "github.com/go-ap/handlers"
	"github.com/mariusor/qstring"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Hash
type Hash string

// String returns the hash as a string
func (h Hash) String() string {
	return string(h)
}

const (
	// ActorsType is a constant that represents the URL path for the local actors collection.
	// It is used as the parent for all To ObjectIDs
	ActorsType = h.CollectionType("actors")
	// ActivitiesType is a constant that represents the URL path for the local activities collection
	// It is used as the parent for all Activity ObjectIDs
	ActivitiesType = h.CollectionType("activities")
	// ObjectsType is a constant that represents the URL path for the local objects collection
	// It is used as the parent for all non To, non Activity Object ObjectIDs
	ObjectsType = h.CollectionType("objects")
)

var validActivityCollection = []h.CollectionType{
	ActorsType,
	ActivitiesType,
	ObjectsType,
}

func getValidActivityCollection(typ string) h.CollectionType {
	for _, t := range validActivityCollection {
		if strings.ToLower(typ) == string(t) {
			return t
		}
	}
	return h.Unknown
}

// ValidActivityCollection shows if the current ActivityPub end-point type is a valid one for handling Activities
func ValidActivityCollection(typ string) bool {
	return getValidActivityCollection(typ) != h.Unknown || h.ValidActivityCollection(typ) || h.ValidObjectCollection(typ)
}

// Filters
type Filters struct {
	baseURL       as.IRI                     `qstring:"-"`
	Name          []string                   `qstring:"name,omitempty"`
	Authenticated *auth.Person               `qstring:"-"`
	To            as.Actor                   `qstring:"-"`
	Author        as.Actor                   `qstring:"-"`
	Parent        as.Actor                   `qstring:"-"`
	IRI           as.IRI                     `qstring:"-"`
	Collection    h.CollectionType           `qstring:"-"`
	URL           as.IRIs                    `qstring:"url,omitempty"`
	MedTypes      []as.MimeType              `qstring:"mediaType,omitempty"`
	Aud           as.IRIs                    `qstring:"-"`
	Key           []Hash                     `qstring:"hash,omitempty"`
	ItemKey       []Hash                     `qstring:"itemHash,omitempty"`
	ObjectKey     []Hash                     `qstring:"object,omitempty"`
	Type          as.ActivityVocabularyTypes `qstring:"type,omitempty"`
	AttrTo        []Hash                     `qstring:"attributedTo,omitempty"`
	InReplTo      []Hash                     `qstring:"inReplyTo,omitempty"`
	OP            []Hash                     `qstring:"context,omitempty"`
	FollowedBy    []Hash                     `qstring:"followedBy,omitempty"` // todo(marius): not really used
	OlderThan     time.Time                  `qstring:"olderThan,omitempty"`
	NewerThan     time.Time                  `qstring:"newerThan,omitempty"`
	CurPage       uint                       `qstring:"page,omitempty"`
	MaxItems      uint                       `qstring:"maxItems,omitempty"`
}

// Types returns a list of ActivityVocabularyTypes to filter against
func (f Filters) Types() as.ActivityVocabularyTypes {
	return f.Type
}

// Context returns a list of ActivityVocabularyTypes to filter against
func (f Filters) Context() as.IRIs {
	ret := make(as.IRIs, 0)
	for _, k := range f.OP {
		// TODO(marius): This piece of logic should be moved to loading the filters
		if k == "" || k == "0" {
			// for empty context we give it a generic filter to skip all objects that have context
			ret = append(ret, as.PublicNS)
			return ret
		}
		var iri as.IRI
		if u, err := url.Parse(string(k)); err == nil {
			iri = as.IRI(u.String())
		} else {
			iri = as.IRI(fmt.Sprintf("%s/%s/%s", f.baseURL, ObjectsType, k))
		}
		if !ret.Contains(iri) {
			ret = append(ret, iri)
		}
	}
	return ret
}

// IRIs returns a list of IRIs to filter against
func (f Filters) IRIs() as.IRIs {
	ret := make(as.IRIs, len(f.ItemKey))
	for i, k := range f.ItemKey {
		ret[i] = as.IRI(k)
	}
	return ret
}

// GetLink returns a list of IRIs to filter against
func (f Filters) GetLink() as.IRI {
	return f.IRI
}

// TODO(marius): move this somewhere else. Or replace it with something that makes more sense.
var Secure = false

func copyActivityFilters(dst *Filters, src Filters) {
	dst.Type = src.Type
	dst.Key = src.Key
	dst.ItemKey = src.ItemKey
	dst.Type = src.Type
	dst.AttrTo = src.AttrTo
	dst.FollowedBy = src.FollowedBy
	dst.OlderThan = src.OlderThan
	dst.NewerThan = src.NewerThan
	dst.CurPage = src.CurPage
	dst.MaxItems = src.MaxItems
}

// Page
func (f Filters) Page() uint {
	return f.CurPage
}

// Count
func (f Filters) Count() uint {
	if f.MaxItems > 0 {
		return f.MaxItems
	}
	return MaxItems
}

const MaxItems = 100

var ErrNotFound = func(s string) error {
	return errors.Newf(fmt.Sprintf("%s not found", s))
}

// TODO(marius): this function also exists in app/filters package
func reqURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL.Path)
}

// TODO(marius): this function also exists in app/filters package
func reqBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}

// FromRequest loads the filters we use for generating storage queries from the HTTP request
func FromRequest(r *http.Request) (*Filters, error) {
	f := Filters{}
	if err := qstring.Unmarshal(r.URL.Query(), &f); err != nil {
		return nil, err
	}
	f.Collection = h.Typer.Type(r)
	f.baseURL = as.IRI(reqBaseURL(r))
	f.IRI = as.IRI(reqURL(r))

	if f.MaxItems > MaxItems {
		f.MaxItems = MaxItems
	}

	return &f, nil
}

// Audience returns a filter for audience members.
// This is important for filtering out objects that don't have a public audience.
func (f Filters) Audience() as.IRIs {
	col := make(as.IRIs, 0)
	for _, iri := range f.Aud {
		rr := as.IRI(fmt.Sprintf("%s/%s/%s", f.baseURL, ActorsType, iri))
		if !col.Contains(rr) {
			col = append(col, rr)
		}
	}
	if f.Authenticated != nil && !col.Contains(f.Authenticated.GetLink()) {
		col = append(col, f.Authenticated.GetLink())
	}
	if !col.Contains(as.PublicNS) {
		col = append(col, as.PublicNS)
	}
	return col
}

func (f Filters) Names() []string {
	return f.Name
}

func (f Filters) AttributedTo() as.IRIs {
	col := make(as.IRIs, len(f.AttrTo))
	for k, iri := range f.AttrTo {
		col[k] = as.IRI(fmt.Sprintf("%s/%s/%s", f.baseURL, ActorsType, iri))
	}
	return col
}

func (f Filters) InReplyTo() as.IRIs {
	col := make(as.IRIs, len(f.InReplTo))
	for k, iri := range f.InReplTo {
		// TODO(marius): This piece of logic should be moved to loading the filters
		if iri == "" || iri == "0" {
			// for empty context we give it a generic filter to skip all objects that have context
			col[k] = as.PublicNS
			return col
		}
		col[k] = as.IRI(fmt.Sprintf("%s/%s/%s", f.baseURL, ObjectsType, iri))
	}
	return col
}

func (f Filters) MediaTypes() []as.MimeType {
	return f.MedTypes
}

func (f Filters) URLs() as.IRIs {
	return f.URL
}

func (f Filters) Actors() as.IRIs {
	return nil
}
func (f Filters) Objects() as.IRIs {
	ret := make(as.IRIs, 0)
	for _, k := range f.ObjectKey {
		// TODO(marius): This piece of logic should be moved to loading the filters
		var iri as.IRI
		if u, err := url.Parse(string(k)); err == nil && u.Host != "" && u.Scheme != "" {
			iri = as.IRI(u.String())
		} else {
			iri = as.IRI(fmt.Sprintf("%s/%s/%s", f.baseURL, ObjectsType, k))
		}
		if !ret.Contains(iri) {
			ret = append(ret, iri)
		}
	}
	return ret
}

func (f Filters) Targets() as.IRIs {
	return nil
}
