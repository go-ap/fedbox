package activitypub

import (
	"fmt"
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	h "github.com/go-ap/handlers"
	"github.com/mariusor/qstring"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

// Hash
type Hash string

// String returns the hash as a string
func (h Hash) String() string {
	return string(h)
}

// String returns the hash as a string
func (h Hash) Matches(i pub.IRI) bool {
	return path.Base(i.String()) == string(h)
}

const (
	// ActorsType is a constant that represents the URL path for the local actors collection.
	// It is used as the parent for all To IDs
	ActorsType = h.CollectionType("actors")
	// ActivitiesType is a constant that represents the URL path for the local activities collection
	// It is used as the parent for all Activity IDs
	ActivitiesType = h.CollectionType("activities")
	// ObjectsType is a constant that represents the URL path for the local objects collection
	// It is used as the parent for all non To, non Activity Object IDs
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
	baseURL       pub.IRI                     `qstring:"-"`
	Name          []string                    `qstring:"name,omitempty"`
	Cont          []string                    `qstring:"content,omitempty"`
	Authenticated *pub.Actor                  `qstring:"-"`
	To            *pub.Actor                  `qstring:"-"`
	Author        *pub.Actor                  `qstring:"-"`
	Parent        *pub.Actor                  `qstring:"-"`
	IRI           pub.IRI                     `qstring:"-"`
	Collection    h.CollectionType            `qstring:"-"`
	URL           pub.IRIs                    `qstring:"url,omitempty"`
	MedTypes      []pub.MimeType              `qstring:"mediaType,omitempty"`
	Aud           pub.IRIs                    `qstring:"recipients,omitempty"`
	Key           []Hash                      `qstring:"-"`
	ItemKey       []Hash                      `qstring:"iri,omitempty"`
	ObjectKey     []Hash                      `qstring:"object,omitempty"`
	ActorKey      []Hash                      `qstring:"actor,omitempty"`
	TargetKey     []Hash                      `qstring:"target,omitempty"`
	Type          pub.ActivityVocabularyTypes `qstring:"type,omitempty"`
	AttrTo        []Hash                      `qstring:"attributedTo,omitempty"`
	InReplTo      []Hash                      `qstring:"inReplyTo,omitempty"`
	OP            []Hash                      `qstring:"context,omitempty"`
	FollowedBy    []Hash                      `qstring:"followedBy,omitempty"` // todo(marius): not really used
	OlderThan     time.Time                   `qstring:"olderThan,omitempty"`
	NewerThan     time.Time                   `qstring:"newerThan,omitempty"`
	Prev          Hash                        `qstring:"before,omitempty"`
	Next          Hash                        `qstring:"after,omitempty"`
	CurPage       uint                        `qstring:"page,omitempty"`
	MaxItems      uint                        `qstring:"maxItems,omitempty"`
}

func NewFilter(s string) Filters {
	return Filters{
		baseURL: pub.IRI(s),
	}
}

// Types returns a list of ActivityVocabularyTypes to filter against
func (f Filters) Types() pub.ActivityVocabularyTypes {
	return f.Type
}

const absentValue = "-"

var AbsentIRI = pub.IRIs{pub.IRI(absentValue)}
var AbsentHash = []Hash{Hash(absentValue)}

// Context returns a list of ActivityVocabularyTypes to filter against
func (f Filters) Context() pub.IRIs {
	ret := make(pub.IRIs, 0)
	for _, k := range f.OP {
		// TODO(marius): This piece of logic should be moved to loading the filters
		if matchAbsent(k) {
			// for empty context we give it a generic filter to skip all objects that have context
			return AbsentIRI
		}
		var iri pub.IRI
		if u, ok := validURL(string(k)); ok {
			iri = pub.IRI(u.String())
		} else {
			iri = pub.IRI(fmt.Sprintf("%s/%s/%s", f.baseURL, ObjectsType, k))
		}
		if !ret.Contains(iri) {
			ret = append(ret, iri)
		}
	}
	return ret
}

// IRIs returns a list of IRIs to filter against
func (f Filters) IRIs() pub.IRIs {
	ret := make(pub.IRIs, len(f.ItemKey))
	for i, k := range f.ItemKey {
		var iri pub.IRI
		if u, ok := validURL(string(k)); ok {
			iri = pub.IRI(u.String())
		} else {
			iri = pub.IRI(fmt.Sprintf("%s/%s/%s", f.baseURL, f.Collection, k))
		}
		if !ret.Contains(iri) {
			ret[i] = pub.IRI(iri)
		}
	}
	return ret
}

// GetLink returns a list of IRIs to filter against
func (f Filters) GetLink() pub.IRI {
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

// Page
func (f Filters) Before() Hash {
	return f.Prev
}

// Page
func (f Filters) After() Hash {
	return f.Next
}

// Count
func (f Filters) Count() uint {
	return f.MaxItems
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
func reqBaseURL(r *url.URL) string {
	return fmt.Sprintf("%s://%s", r.Scheme, r.Host)
}

// FromRequest loads the filters we use for generating storage queries from the HTTP request
func FromRequest(r *http.Request) (*Filters, error) {
	f := Filters{}
	if err := qstring.Unmarshal(r.URL.Query(), &f); err != nil {
		return nil, err
	}
	f.Collection = h.Typer.Type(r)
	if len(f.IRI) == 0 {
		f.IRI = pub.IRI(reqURL(r))
	}
	i, _ := f.IRI.URL()
	f.baseURL = pub.IRI(reqBaseURL(i))

	if f.MaxItems > MaxItems {
		f.MaxItems = MaxItems
	}

	return &f, nil
}

// Audience returns a filter for audience members.
// This is important for filtering out objects that don't have a public audience.
func (f Filters) Audience() pub.IRIs {
	col := make(pub.IRIs, 0)
	for _, iri := range f.Aud {
		if iri == pub.EmptyIRI || iri == "0" || iri == absentValue {
			iri = pub.PublicNS
		} else {
			iri = pub.IRI(fmt.Sprintf("%s/%s/%s", f.baseURL, ActorsType, iri))
		}
		if !col.Contains(iri) {
			col = append(col, iri)
		}
	}
	if f.Authenticated != nil && !col.Contains(f.Authenticated.GetLink()) {
		col = append(col, f.Authenticated.GetLink())
	}
	if !col.Contains(pub.PublicNS) {
		col = append(col, pub.PublicNS)
	}
	return col
}

func (f Filters) Names() []string {
	return f.Name
}
func (f Filters) Content() []string {
	return f.Cont
}

func validURL(s string) (*url.URL, bool) {
	u, err := url.Parse(s)
	return u, err == nil && u.Host != "" && u.Scheme != ""
}

func (f Filters) AttributedTo() pub.IRIs {
	col := make(pub.IRIs, len(f.AttrTo))
	for k, iri := range f.AttrTo {
		// TODO(marius): This piece of logic should be moved to loading the filters
		if matchAbsent(iri) {
			// for empty context we give it a generic filter to skip all objects that have context
			return AbsentIRI
		}
		if _, ok := validURL(iri.String()); ok {
			col[k] = pub.IRI(iri)
		} else {
			col[k] = pub.IRI(fmt.Sprintf("%s/%s/%s", f.baseURL, ActorsType, iri))
		}
	}
	return col
}

func matchAbsent(i fmt.Stringer) bool {
	iri := i.String()
	return iri == "" || iri == "0" || iri == absentValue
}

func (f Filters) InReplyTo() pub.IRIs {
	col := make(pub.IRIs, len(f.InReplTo))
	for k, iri := range f.InReplTo {
		// TODO(marius): This piece of logic should be moved to loading the filters
		if matchAbsent(iri) {
			// for empty context we give it a generic filter to skip all objects that have context
			return AbsentIRI
		}
		if _, ok := validURL(iri.String()); ok {
			col[k] = pub.IRI(iri)
		} else {
			col[k] = pub.IRI(fmt.Sprintf("%s/%s/%s", f.baseURL, ObjectsType, iri))
		}
	}
	return col
}

func (f Filters) MediaTypes() []pub.MimeType {
	return f.MedTypes
}

func (f Filters) URLs() pub.IRIs {
	return f.URL
}

func (f Filters) Actors() pub.IRIs {
	ret := make(pub.IRIs, 0)
	for _, k := range f.ActorKey {
		// TODO(marius): This piece of logic should be moved to loading the filters
		var iri pub.IRI
		if u, ok := validURL(string(k)); ok {
			iri = pub.IRI(u.String())
		} else {
			iri = pub.IRI(fmt.Sprintf("%s/%s/%s", f.baseURL, ActorsType, k))
		}
		if !ret.Contains(iri) {
			ret = append(ret, iri)
		}
	}
	return ret
}

func (f Filters) Objects() pub.IRIs {
	ret := make(pub.IRIs, 0)
	for _, k := range f.ObjectKey {
		// TODO(marius): This piece of logic should be moved to loading the filters
		var iri pub.IRI
		if u, ok := validURL(string(k)); ok {
			iri = pub.IRI(u.String())
		} else {
			iri = pub.IRI(fmt.Sprintf("%s/%s/%s", f.baseURL, ObjectsType, k))
		}
		if !ret.Contains(iri) {
			ret = append(ret, iri)
		}
	}
	return ret
}

func (f Filters) Targets() pub.IRIs {
	ret := make(pub.IRIs, 0)
	for _, k := range f.TargetKey {
		// TODO(marius): This piece of logic should be moved to loading the filters
		var iris pub.IRIs
		if u, ok := validURL(string(k)); ok {
			iris = pub.IRIs{pub.IRI(u.String())}
		} else {
			// FIXME(marius): we don't really know which type this is
			iris = pub.IRIs{
				pub.IRI(fmt.Sprintf("%s/%s/%s", f.baseURL, ObjectsType, k)),
				pub.IRI(fmt.Sprintf("%s/%s/%s", f.baseURL, ActorsType, k)),
				pub.IRI(fmt.Sprintf("%s/%s/%s", f.baseURL, ActivitiesType, k)),
			}
		}
		for _, iri := range iris {
			if !ret.Contains(iri) {
				ret = append(ret, iri)
			}
		}
	}
	return ret
}

func filterObject(it pub.Item, ff Filters) (bool, pub.Item) {
	keep := true
	pub.OnObject(it, func(ob *pub.Object) error {
		if !filterNaturalLanguageValuesExactMatch(ff.Names(), ob.Name) {
			keep = false
			return nil
		}
		if !filterNaturalLanguageValuesSubstring(ff.Content(), ob.Content, ob.Summary) {
			keep = false
			return nil
		}
		if !filterURLs(ff.URLs(), ob) {
			keep = false
			return nil
		}
		if !filterWithAbsent(ff.Context(), ob.Context) {
			keep = false
			return nil
		}
		if !filterWithAbsent(ff.AttributedTo(), ob.AttributedTo) {
			keep = false
			return nil
		}
		if !filterWithAbsent(ff.InReplyTo(), ob.InReplyTo) {
			keep = false
			return nil
		}
		if !filterAudience(ff.Audience(), ob.Recipients(), pub.ItemCollection{ob.AttributedTo}) {
			keep = false
			return nil
		}
		if !filterMediaTypes(ff.MediaTypes(), ob.MediaType) {
			keep = false
			return nil
		}
		return nil
	})
	return keep, it
}

func filterActivity(it pub.Item, ff Filters) (bool, pub.Item) {
	keep := true
	pub.OnActivity(it, func(act *pub.Activity) error {
		if ok, _ := filterObject(act, ff); !ok {
			keep = false
			return nil
		}
		if !filterItem(ff.Actors(), act.Actor) {
			keep = false
			return nil
		}
		if !filterItem(ff.Objects(), act.Object) {
			keep = false
			return nil
		}
		if !filterItem(ff.Targets(), act.Target) {
			keep = false
			return nil
		}
		return nil
	})
	return keep, it
}

func filterActor(it pub.Item, ff Filters) (bool, pub.Item) {
	keep := true
	pub.OnActor(it, func(ob *pub.Actor) error {
		names := ff.Names()
		if len(names) > 0 && !filterNaturalLanguageValuesExactMatch(names, ob.Name, ob.PreferredUsername) {
			keep = false
			return nil
		}
		if !filterItem(ff.URLs(), ob) {
			keep = false
			return nil
		}
		if !filterWithAbsent(ff.Context(), ob.Context) {
			keep = false
			return nil
		}
		// TODO(marius): this needs to be moved in handling an item collection for inReplyTo
		if !filterWithAbsent(ff.Context(), ob.InReplyTo) {
			keep = false
			return nil
		}
		if !filterItem(ff.AttributedTo(), ob.AttributedTo) {
			keep = false
			return nil
		}
		if !filterItemCollections(ff.InReplyTo(), ob.InReplyTo) {
			keep = false
			return nil
		}
		if !filterAudience(ff.Audience(), ob.Recipients(), pub.ItemCollection{ob.AttributedTo}) {
			keep = false
			return nil
		}
		if !filterMediaTypes(ff.MediaTypes(), ob.MediaType) {
			keep = false
			return nil
		}
		return nil
	})
	return keep, it
}

func filterNaturalLanguageValuesSubstring(filters []string, valArr ...pub.NaturalLanguageValues) bool {
	keep := true
	if len(filters) > 0 {
		keep = false
	}
	for _, filter := range filters {
		for _, langValues := range valArr {
			for _, langValue := range langValues {
				if strings.Contains(strings.ToLower(langValue.Value), strings.ToLower(filter)) {
					keep = true
					break
				}
				if keep {
					break
				}
			}
		}
	}
	return keep
}

func filterNaturalLanguageValuesExactMatch(filters []string, valArr ...pub.NaturalLanguageValues) bool {
	keep := true
	if len(filters) > 0 {
		keep = false
	}
	for _, filter := range filters {
		for _, langValues := range valArr {
			for _, langValue := range langValues {
				if strings.ToLower(langValue.Value) == strings.ToLower(filter) {
					keep = true
					break
				}
				if keep {
					break
				}
			}
		}
	}
	return keep
}

func filterItems(filters pub.IRIs, items ...pub.Item) bool {
	if len(filters) == 0 {
		return true
	}
	if hasAbsentFilter(filters) && filterAbsent(filters, items...) {
		return true
	}
	for _, it := range items {
		if it == nil {
			continue
		}
		lnk := it.GetLink()
		if filters.Contains(lnk) {
			return true
		}
	}
	return false
}

func filterAudience(filters pub.IRIs, colArr ...pub.ItemCollection) bool {
	if len(filters) == 0 {
		return true
	}
	allItems := make(pub.ItemCollection, 0)
	for _, items := range colArr {
		for _, it := range items {
			if it != nil {
				allItems = append(allItems, it)
			}
		}
	}
	allItems, _ = pub.ItemCollectionDeduplication(&allItems)
	return filterItems(filters, allItems...)
}

func filterItemCollections(filters pub.IRIs, colArr ...pub.Item) bool {
	if len(filters) == 0 {
		return true
	}

	allItems := make(pub.ItemCollection, 0)
	for _, col := range colArr {
		if col == nil {
			continue
		}
		if col.IsCollection() {
			pub.OnCollectionIntf(col, func(c pub.CollectionInterface) error {
				for _, it := range c.Collection() {
					if it != nil {
						allItems = append(allItems, it)
					}
				}
				return nil
			})
		} else {
			allItems = append(allItems, col)
		}
	}
	pub.ItemCollectionDeduplication(&allItems)
	return filterItems(filters, allItems...)
}

func hasAbsentFilter(filters pub.IRIs) bool {
	if len(filters) != 1 {
		return false
	}
	return filters[0] == AbsentIRI[0]
}

// filterAbsent is used when searching that the incoming items collection is empty
func filterAbsent(filters pub.IRIs, items ...pub.Item) bool {
	if filters[0] == AbsentIRI[0] {
		if len(items) == 0 {
			return true
		}
		for _, it := range items {
			if it == nil {
				continue
			}
			if it.IsCollection() {
				result := false
				pub.OnCollectionIntf(it, func(c pub.CollectionInterface) error {
					for _, it := range c.Collection() {
						if it == nil {
							continue
						}
						if it != nil && it.GetLink() == pub.PublicNS { // FIXME(marius): this is kinda ugly
							result = true
							return nil
						}
					}
					return nil
				})
				return result
			}
			if it != nil && it.GetLink() != pub.PublicNS { // FIXME(marius): this is kinda ugly
				return false
			}
		}
	}
	return true
}

func filterWithAbsent(filters pub.IRIs, items ...pub.Item) bool {
	if len(filters) == 0 {
		return true
	}
	if hasAbsentFilter(filters) && filterAbsent(filters, items...) {
		return true
	}
	keep := true
	for _, it := range items {
		keep = filterItem(filters, it)
	}
	return keep
}

func filterItem(filters pub.IRIs, it pub.Item) bool {
	keep := true
	if len(filters) > 0 {
		if it == nil {
			return false
		}
		if c, ok := it.(pub.ItemCollection); ok {
			return filterItems(filters, c...)
		} else {
			keep = filters.Contains(it.GetLink())
		}
	}
	return keep
}

func filterURLs(filters pub.IRIs, it pub.Item) bool {
	if len(filters) == 0 {
		return true
	}
	keep := false
	if it == nil {
		return false
	}
	var url string
	switch ob := it.(type) {
	case pub.Page:
		url = ob.URL.GetLink().String()
	case *pub.Page:
		url = ob.URL.GetLink().String()
	}
	if url == "" {
		pub.OnObject(it, func(o *pub.Object) error {
			if o.URL != nil {
				url = o.URL.GetLink().String()
			}
			return nil
		})
	}
	for _, filter := range filters {
		if strings.Contains(url, filter.String()) {
			keep = true
			break
		}
	}
	return keep
}

func filterMediaTypes(medTypes []pub.MimeType, typ pub.MimeType) bool {
	keep := true
	if len(medTypes) > 0 {
		exists := false
		for _, filter := range medTypes {
			if filter == typ {
				exists = true
			}
		}
		if !exists {
			keep = false
		}
	}
	return keep
}

type CollectionFilterer interface {
	FilterCollection(col pub.ItemCollection) (pub.ItemCollection, int)
}

type ItemMatcher interface {
	ItemMatches(it pub.Item) bool
}

func (f Filters) FilterCollection(col pub.ItemCollection) (pub.ItemCollection, int) {
	if len(col) == 0 {
		return col, 0
	}
	new := make(pub.ItemCollection, len(col))
	for _, it := range col {
		if f.ItemMatches(it) {
			new = append(new, it)
		}
	}
	col = new
	return nil, 0
}

// ugly hack to check if the current filter f.IRI property is a collection or an object
func iriIsObject(iri pub.IRI) bool {
	base := path.Base(iri.String())
	return !ValidActivityCollection(base)
}

// ItemMatches
func (f Filters) ItemMatches(it pub.Item) bool {
	if it == nil {
		return false
	}
	iris := f.IRIs()
	// FIXME(marius): the Contains method returns true for the case where IRIs is empty, we don't want that
	if len(iris) > 0 && !iris.Contains(it.GetLink()) {
		return false
	}
	types := f.Types()
	// FIXME(marius): this does not cover case insensitivity
	if len(types) > 0 && !types.Contains(it.GetType()) {
		return false
	}
	iri := f.GetLink()
	if len(iri) > 0 && iriIsObject(iri) {
		if !iri.Contains(it.GetLink(), false) {
			return false
		}
	}
	var valid bool
	if pub.ActivityTypes.Contains(it.GetType()) || pub.IntransitiveActivityTypes.Contains(it.GetType()) {
		valid, _ = filterActivity(it, f)
	} else if pub.ActorTypes.Contains(it.GetType()) {
		valid, _ = filterActor(it, f)
	} else {
		valid, _ = filterObject(it, f)
	}
	return valid
}
