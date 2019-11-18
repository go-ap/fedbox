package activitypub

import (
	"fmt"
	"github.com/go-ap/activitypub"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/auth"
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
	Cont          []string                   `qstring:"content,omitempty"`
	Authenticated *auth.Person               `qstring:"-"`
	To            as.Actor                   `qstring:"-"`
	Author        as.Actor                   `qstring:"-"`
	Parent        as.Actor                   `qstring:"-"`
	IRI           as.IRI                     `qstring:"-"`
	Collection    h.CollectionType           `qstring:"-"`
	URL           as.IRIs                    `qstring:"url,omitempty"`
	MedTypes      []as.MimeType              `qstring:"mediaType,omitempty"`
	Aud           as.IRIs                    `qstring:"-"`
	Key           []Hash                     `qstring:"-"`
	ItemKey       []Hash                     `qstring:"-"`
	ObjectKey     []Hash                     `qstring:"-"`
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

func NewFilter(s string) Filters {
	return Filters{
		baseURL: as.IRI(s),
	}
}

// Types returns a list of ActivityVocabularyTypes to filter against
func (f Filters) Types() as.ActivityVocabularyTypes {
	return f.Type
}

const absentValue = "-"

var AbsentIRI = as.IRIs{as.IRI(absentValue)}
var AbsentHash = []Hash{Hash(absentValue)}

// Context returns a list of ActivityVocabularyTypes to filter against
func (f Filters) Context() as.IRIs {
	ret := make(as.IRIs, 0)
	for _, k := range f.OP {
		// TODO(marius): This piece of logic should be moved to loading the filters
		if matchAbsent(k) {
			// for empty context we give it a generic filter to skip all objects that have context
			return AbsentIRI
		}
		var iri as.IRI
		if u, ok := validURL(string(k)); ok {
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
		f.IRI = as.IRI(reqURL(r))
	}
	i, _ := f.IRI.URL()
	f.baseURL = as.IRI(reqBaseURL(i))

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
func (f Filters) Content() []string {
	return f.Cont
}

func validURL(s string) (*url.URL, bool) {
	u, err := url.Parse(s)
	return u, err == nil && u.Host != "" && u.Scheme != ""
}

func (f Filters) AttributedTo() as.IRIs {
	col := make(as.IRIs, len(f.AttrTo))
	for k, iri := range f.AttrTo {
		// TODO(marius): This piece of logic should be moved to loading the filters
		if matchAbsent(iri) {
			// for empty context we give it a generic filter to skip all objects that have context
			return AbsentIRI
		}
		if _, ok := validURL(iri.String()); ok {
			col[k] = as.IRI(iri)
		} else {
			col[k] = as.IRI(fmt.Sprintf("%s/%s/%s", f.baseURL, ActorsType, iri))
		}
	}
	return col
}

func matchAbsent(i fmt.Stringer) bool {
	iri := i.String()
	return iri == "" || iri == "0" || iri == absentValue
}

func (f Filters) InReplyTo() as.IRIs {
	col := make(as.IRIs, len(f.InReplTo))
	for k, iri := range f.InReplTo {
		// TODO(marius): This piece of logic should be moved to loading the filters
		if matchAbsent(iri) {
			// for empty context we give it a generic filter to skip all objects that have context
			return AbsentIRI
		}
		if _, ok := validURL(iri.String()); ok {
			col[k] = as.IRI(iri)
		} else {
			col[k] = as.IRI(fmt.Sprintf("%s/%s/%s", f.baseURL, ObjectsType, iri))
		}
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
		if u, ok := validURL(string(k)); ok {
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

func filterObject(it as.Item, ff Filters) (bool, as.Item) {
	keep := true
	activitypub.OnObject(it, func(ob *activitypub.Object) error {
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
		if !filterWithAbsent(ff.InReplyTo(), ob.InReplyTo...) {
			keep = false
			return nil
		}
		if !filterAudience(ff.Audience(), ob.Recipients(), as.ItemCollection{ob.AttributedTo}) {
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

func filterActivity(it as.Item, ff Filters) (bool, as.Item) {
	keep := true
	activitypub.OnActivity(it, func(act *as.Activity) error {
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

func filterActor(it as.Item, ff Filters) (bool, as.Item) {
	keep := true
	auth.OnPerson(it, func(ob *auth.Person) error {
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
		if !filterWithAbsent(ff.Context(), ob.InReplyTo...) {
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
		if !filterAudience(ff.Audience(), ob.Recipients(), as.ItemCollection{ob.AttributedTo}) {
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

func filterNaturalLanguageValuesSubstring(filters []string, valArr ...as.NaturalLanguageValues) bool {
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

func filterNaturalLanguageValuesExactMatch(filters []string, valArr ...as.NaturalLanguageValues) bool {
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

func filterItems(filters as.IRIs, items ...as.Item) bool {
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

func filterAudience(filters as.IRIs, colArr ...as.ItemCollection) bool {
	if len(filters) == 0 {
		return true
	}
	allItems := make(as.ItemCollection, 0)
	for _, items := range colArr {
		for _, it := range items {
			if it != nil {
				allItems = append(allItems, it)
			}
		}
	}
	allItems, _ = as.ItemCollectionDeduplication(&allItems)
	return filterItems(filters, allItems...)
}

func filterItemCollections(filters as.IRIs, colArr ...as.ItemCollection) bool {
	if len(filters) == 0 {
		return true
	}
	allItems := make(as.ItemCollection, 0)
	for _, items := range colArr {
		for _, it := range items {
			if it != nil {
				allItems = append(allItems, it)
			}
		}
	}
	as.ItemCollectionDeduplication(&allItems)
	return filterItems(filters, allItems...)
}

func hasAbsentFilter(filters as.IRIs) bool {
	if len(filters) != 1 {
		return false
	}
	return filters[0] == AbsentIRI[0]
}

// filterAbsent is used when searching that the incoming items collection is empty
func filterAbsent(filters as.IRIs, items ...as.Item) bool {
	if filters[0] == AbsentIRI[0] {
		if len(items) == 0 {
			return true
		}
		for _, it := range items {
			if it != nil && it.GetLink() != as.PublicNS { // FIXME(marius): this is kinda ugly
				return false
			}
		}
	}
	return true
}

func filterWithAbsent(filters as.IRIs, items ...as.Item) bool {
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

func filterItem(filters as.IRIs, it as.Item) bool {
	keep := true
	if len(filters) > 0 {
		if it == nil {
			return false
		}
		keep = filters.Contains(it.GetLink())
	}
	return keep
}

func filterURLs(filters as.IRIs, it as.Item) bool {
	if len(filters) == 0 {
		return true
	}
	keep := false
	if it == nil {
		return false
	}
	var url string
	switch ob := it.(type) {
	case as.Page:
		url = ob.URL.GetLink().String()
	case *as.Page:
		url = ob.URL.GetLink().String()
	}
	if url == "" {
		activitypub.OnObject(it, func(o *activitypub.Object) error {
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

func filterMediaTypes(medTypes []as.MimeType, typ as.MimeType) bool {
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
	FilterCollection(col as.ItemCollection) (as.ItemCollection, int)
}

type ItemMatcher interface {
	ItemMatches(it as.Item) bool
}

func (f Filters) FilterCollection(col as.ItemCollection) (as.ItemCollection, int) {
	if len(col) == 0 {
		return col, 0
	}
	new := make(as.ItemCollection, len(col))
	for _, it := range col {
		if f.ItemMatches(it) {
			new = append(new, it)
		}
	}
	col = new
	return nil, 0
}

// ugly hack to check if the current filter f.IRI property is a collection or an object
func iriIsObject(iri as.IRI) bool {
	base := path.Base(iri.String())
	return !ValidActivityCollection(base)
}

// ItemMatches
func (f Filters) ItemMatches(it as.Item) bool {
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
	var valid bool
	if as.ActivityTypes.Contains(it.GetType()) {
		valid, _ = filterActivity(it, f)
	} else if as.IntransitiveActivityTypes.Contains(it.GetType()) {
		// FIXME(marius): this does not work
		valid, _ = filterActivity(it, f)
	} else if as.ActorTypes.Contains(it.GetType()) {
		valid, _ = filterActor(it, f)
	} else {
		valid, _ = filterObject(it, f)
	}
	iri := f.GetLink()
	if len(iri) > 0 && iriIsObject(iri) {
		itIRI := it.GetLink()
		valid = iri == itIRI || iri.String() == fmt.Sprintf("%s/", itIRI)
	}

	return valid
}
