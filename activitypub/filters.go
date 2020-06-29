package activitypub

import (
	"fmt"
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	h "github.com/go-ap/handlers"
	s "github.com/go-ap/storage"
	"github.com/mariusor/qstring"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

// Hash
type Hash string

type CompStr = qstring.ComparativeString
type CompStrs []CompStr

func StringEquals(s string) CompStr {
	return CompStr{Str: s}
}
func StringLike(s string) CompStr {
	return CompStr{Operator: "~", Str: s}
}
func StringDifferent(s string) CompStr {
	return CompStr{Operator: "!", Str: s}
}

func (cs CompStrs) Contains(f CompStr) bool {
	for _, c := range cs {
		if c.Str == f.Str {
			return true
		}
	}
	return false
}

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

	// BlockedType is an internally used collection, to store a list of actors the actor has blocked
	BlockedType = h.CollectionType("blocked")
	// IgnoredType is an internally used collection, to store a list of actors the actor has ignored
	IgnoredType = h.CollectionType("ignored")
)

// @todo(marius): here we need a better separation between the collections which are exposed in the HTTP API
//   (activities,actors,objects) and the ones that are internal (blocked,ignored)
var (
	FedboxCollections = h.CollectionTypes{
		ActivitiesType,
		ActorsType,
		ObjectsType,
		BlockedType,
		IgnoredType,
	}

	validActivityCollection = []h.CollectionType{
		ActivitiesType,
	}

	validObjectCollection = []h.CollectionType{
		ActorsType,
		ObjectsType,
	}
)

func getValidActivityCollection(typ h.CollectionType) h.CollectionType {
	for _, t := range validActivityCollection {
		if strings.ToLower(string(typ)) == string(t) {
			return t
		}
	}
	return h.Unknown
}

func getValidObjectCollection(typ h.CollectionType) h.CollectionType {
	for _, t := range validObjectCollection {
		if strings.ToLower(string(typ)) == string(t) {
			return t
		}
	}
	return h.Unknown
}

// ValidCollection shows if the current ActivityPub end-point type is a valid collection
func ValidCollection(typ h.CollectionType) bool {
	return ValidActivityCollection(typ) || ValidObjectCollection(typ)
}

// ValidActivityCollection shows if the current ActivityPub end-point type is a valid collection for handling Activities
func ValidActivityCollection(typ h.CollectionType) bool {
	return getValidActivityCollection(typ) != h.Unknown || h.ValidActivityCollection(typ)
}

// ValidObjectCollection shows if the current ActivityPub end-point type is a valid collection for handling Objects
func ValidObjectCollection(typ h.CollectionType) bool {
	return getValidObjectCollection(typ) != h.Unknown || h.ValidObjectCollection(typ)
}

// Filters
// TODO(marius) we can make some small changes so it's not necessary to export this struct
type Filters struct {
	baseURL       pub.IRI                     `qstring:"-"`
	Name          CompStrs                    `qstring:"name,omitempty"`
	Cont          CompStrs                    `qstring:"content,omitempty"`
	Authenticated *pub.Actor                  `qstring:"-"`
	To            *pub.Actor                  `qstring:"-"`
	Author        *pub.Actor                  `qstring:"-"`
	Parent        *pub.Actor                  `qstring:"-"`
	IRI           pub.IRI                     `qstring:"-"`
	Collection    h.CollectionType            `qstring:"-"`
	URL           CompStrs                    `qstring:"url,omitempty"`
	MedTypes      []pub.MimeType              `qstring:"mediaType,omitempty"`
	Aud           CompStrs                    `qstring:"recipients,omitempty"`
	Gen           CompStrs                    `qstring:"generator,omitempty"`
	Key           []Hash                      `qstring:"-"`
	ItemKey       CompStrs                    `qstring:"iri,omitempty"`
	Type          pub.ActivityVocabularyTypes `qstring:"type,omitempty"`
	AttrTo        CompStrs                    `qstring:"attributedTo,omitempty"`
	InReplTo      CompStrs                    `qstring:"inReplyTo,omitempty"`
	OP            CompStrs                    `qstring:"context,omitempty"`
	FollowedBy    []Hash                      `qstring:"followedBy,omitempty"` // todo(marius): not really used
	OlderThan     time.Time                   `qstring:"olderThan,omitempty"`
	NewerThan     time.Time                   `qstring:"newerThan,omitempty"`
	Prev          Hash                        `qstring:"before,omitempty"`
	Next          Hash                        `qstring:"after,omitempty"`
	Object        *Filters                    `qstring:"object,omitempty"`
	Actor         *Filters                    `qstring:"actor,omitempty"`
	Target        *Filters                    `qstring:"target,omitempty"`
	CurPage       uint                        `qstring:"page,omitempty"`
	MaxItems      uint                        `qstring:"maxItems,omitempty"`
}

func ItemKey(keys ...string) filterFn {
	return func(f *Filters) error {
		if len(f.ItemKey) == 0 {
			f.ItemKey = make(CompStrs, 0)
		}
		for _, key := range keys {
			f.ItemKey = append(f.ItemKey, StringEquals(key))
		}
		return nil
	}
}

func IRI(i pub.IRI) filterFn {
	return func(f *Filters) error {
		f.IRI = i
		return nil
	}
}

func Name(names ...string) filterFn {
	return func(f *Filters) error {
		if len(f.Name) == 0 {
			f.Name = make(CompStrs, 0)
		}
		for _, name := range names {
			f.Name = append(f.Name, StringEquals(name))
		}
		return nil
	}
}

func Type(types ...pub.ActivityVocabularyType) filterFn {
	return func(f *Filters) error {
		f.Type = types
		return nil
	}
}

func BaseIRI(iri pub.IRI, col h.CollectionType) filterFn {
	return func(f *Filters) error {
		f.baseURL = iri
		f.Collection = col
		return nil
	}
}

type filterFn func(f *Filters) error

func FiltersNew(filters ...filterFn) *Filters {
	f := &Filters{}
	for _, fn := range filters {
		fn(f)
	}
	return f
}

// Types returns a list of ActivityVocabularyTypes to filter against
func (f Filters) Types() pub.ActivityVocabularyTypes {
	return f.Type
}

const absentValue = "-"

var AbsentIRIs = CompStrs{AbsentIRI}
var AbsentIRI = CompStr{Str: absentValue, Operator: "="}
var AbsentHash = []Hash{Hash(absentValue)}

// Context returns a list of ActivityVocabularyTypes to filter against
func (f Filters) Context() CompStrs {
	ret := make(CompStrs, 0)
	for _, k := range f.OP {
		// TODO(marius): This piece of logic should be moved to loading the filters
		if matchAbsent(k) {
			// for empty context we give it a generic filter to skip all objects that have context
			return AbsentIRIs
		}
		iri := CompStr{}
		if u, ok := validURL(k.Str); ok {
			iri.Str = u.String()
		} else {
			iri.Str = fmt.Sprintf("%s/%s/%s", f.baseURL, ObjectsType, k)
		}
		if !ret.Contains(iri) {
			ret = append(ret, iri)
		}
	}
	return ret
}

func IRIf(f Filters, iri string) string {
	if _, ok := validURL(iri); ok {
		return iri
	}
	col := f.Collection
	if col != ActorsType && col != ActivitiesType && col != ObjectsType {
		if h.ValidObjectCollection(f.Collection) {
			col = ObjectsType
		} else if ValidActivityCollection(f.Collection) {
			col = ActivitiesType
		}
	}
	if len(f.baseURL) > 0 {
		if u, err := url.Parse(f.baseURL.String()); err == nil {
			if len(col) > 0 {
				u.Path = "/" + string(col)
			}
			if len(u.String()) > 0 {
				iri = fmt.Sprintf("%s/%s", u.String(), iri)
			}
		}
	} else if !strings.Contains(iri, string(col)) {
		iri = fmt.Sprintf("/%s/%s", col, iri)
	}
	return iri
}

// IRIs returns a list of IRIs to filter against
func (f Filters) IRIs() CompStrs {
	ret := make(CompStrs, len(f.ItemKey))
	for i, k := range f.ItemKey {
		if k.Operator == "" || k.Operator == "=" {
			k.Str = IRIf(f, k.Str)
		}
		ret[i] = k
	}
	return ret
}

// GetLink returns a list of IRIs to filter against
func (f *Filters) GetLink() pub.IRI {
	if f.IRI != "" {
		return f.IRI
	}
	return f.baseURL.AddPath(string(f.Collection))
}

// TODO(marius): move this somewhere else. Or replace it with something that makes more sense.
var Secure = false

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

const MaxItems uint = 100

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
	f := FiltersNew()
	if err := qstring.Unmarshal(r.URL.Query(), f); err != nil {
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

	if f.Object != nil {
		f.Object.Collection = ObjectsType
		f.Object.baseURL = f.baseURL
	}
	if f.Actor != nil {
		f.Actor.Collection = ActorsType
		f.Actor.baseURL = f.baseURL
	}

	return f, nil
}

// Audience returns a filter for audience members.
// This is important for filtering out objects that don't have a public audience.
func (f Filters) Audience() CompStrs {
	col := make(CompStrs, 0)
	for _, iri := range f.Aud {
		// TODO(marius): This piece of logic should be moved to loading the filters
		if matchAbsent(iri) {
			// for empty context we give it a generic filter to skip all objects that have context
			return AbsentIRIs
		}
		iri.Str = IRIf(f, iri.Str)
		col = append(col, iri)
	}
	if f.Authenticated != nil {
		user := StringEquals(f.Authenticated.GetLink().String())
		if f.Authenticated != nil && !col.Contains(user) {
			col = append(col, user)
		}
	}
	public := StringEquals(pub.PublicNS.String())
	if !col.Contains(public) {
		col = append(col, public)
	}
	return col
}

func (f Filters) Names() CompStrs {
	return f.Name
}

func (f Filters) Content() CompStrs {
	return f.Cont
}

func validURL(s string) (*url.URL, bool) {
	u, err := url.Parse(s)
	return u, err == nil && u.Host != "" && u.Scheme != ""
}

func (f Filters) AttributedTo() CompStrs {
	for k, iri := range f.AttrTo {
		// TODO(marius): This piece of logic should be moved to loading the filters
		if matchAbsent(iri) {
			// for empty context we give it a generic filter to skip all objects that have context
			return AbsentIRIs
		}
		f.Collection = ActorsType
		iri.Str = IRIf(f, iri.Str)
		f.AttrTo[k] = iri
	}
	return f.AttrTo
}

func matchAbsent(i fmt.Stringer) bool {
	iri := i.String()
	return iri == "" || iri == "0" || iri == absentValue
}

func (f Filters) InReplyTo() CompStrs {
	for k, iri := range f.InReplTo {
		// TODO(marius): This piece of logic should be moved to loading the filters
		if matchAbsent(iri) {
			// for empty context we give it a generic filter to skip all objects that have context
			return AbsentIRIs
		}
		f.Collection = ObjectsType
		iri.Str = IRIf(f, iri.Str)
		f.InReplTo[k] = iri
	}
	return f.InReplTo
}

func (f Filters) MediaTypes() []pub.MimeType {
	return f.MedTypes
}

func (f Filters) URLs() CompStrs {
	return f.URL
}

func (f Filters) Generator() CompStrs {
	return f.Gen
}

func (f Filters) Actors() pub.IRIs {
	ret := make(pub.IRIs, 0)
	if f.Actor == nil {
		return nil
	}
	for _, k := range f.Actor.Key {
		// TODO(marius): This piece of logic should be moved to loading the filters
		f.Collection = ActorsType
		iri := pub.IRI(IRIf(f, k.String()))
		if !ret.Contains(iri) {
			ret = append(ret, iri)
		}
	}
	return ret
}

func (f Filters) Objects() pub.IRIs {
	ret := make(pub.IRIs, 0)
	if f.Object == nil {
		return nil
	}
	for _, k := range f.Object.Key {
		// TODO(marius): This piece of logic should be moved to loading the filters
		f.Collection = ObjectsType
		iri := pub.IRI(IRIf(f, k.String()))
		if !ret.Contains(iri) {
			ret = append(ret, iri)
		}
	}
	return ret
}

func (f Filters) Targets() pub.IRIs {
	ret := make(pub.IRIs, 0)
	if f.Target == nil {
		return nil
	}
	for _, k := range f.Target.Key {
		// TODO(marius): This piece of logic should be moved to loading the filters
		var iris pub.IRIs
		if u, ok := validURL(k.String()); ok {
			iris = pub.IRIs{pub.IRI(u.String())}
		} else {
			// FIXME(marius): we don't really know which type this is
			iris = pub.IRIs{
				ObjectsType.IRI(f.baseURL).AddPath(k.String()),
				ActorsType.IRI(f.baseURL).AddPath(k.String()),
				ActivitiesType.IRI(f.baseURL).AddPath(k.String()),
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

func filterObjectNoName(ob *pub.Object, ff *Filters) bool {
	if ff == nil {
		return true
	}
	keep := true
	if !filterNaturalLanguageValues(ff.Content(), ob.Content, ob.Summary) {
		keep = false
	}
	if !filterWithAbsent(ff.Generator(), ob.Generator) {
		keep = false
	}
	if !filterURLs(ff.URLs(), ob) {
		keep = false
	}
	if !filterWithAbsent(ff.Context(), ob.Context, ob.InReplyTo) {
		keep = false
	}
	if !filterWithAbsent(ff.InReplyTo(), ob.InReplyTo) {
		keep = false
	}
	if !filterAudience(ff.Audience(), ob.Recipients(), pub.ItemCollection{ob.AttributedTo}) {
		keep = false
	}
	if !filterMediaTypes(ff.MediaTypes(), ob.MediaType) {
		keep = false
	}
	return keep
}

func filterObject(it pub.Item, ff *Filters) (bool, pub.Item) {
	if ff == nil {
		return true, it
	}
	keep := true
	pub.OnObject(it, func(ob *pub.Object) error {
		if !filterNaturalLanguageValues(ff.Names(), ob.Name) {
			keep = false
			return nil
		}
		keep = filterObjectNoName(ob, ff)
		return nil
	})
	return keep, it
}

func filterActivity(it pub.Item, ff *Filters) (bool, pub.Item) {
	if ff == nil {
		return true, it
	}
	keep := true
	pub.OnActivity(it, func(act *pub.Activity) error {
		if ok, _ := filterObject(act, ff); !ok {
			keep = false
			return nil
		}
		if !ff.Actor.ItemMatches(act.Actor) {
			keep = false
			return nil
		}
		if !ff.Object.ItemMatches(act.Object) {
			keep = false
			return nil
		}
		if !ff.Target.ItemMatches(act.Target) {
			keep = false
			return nil
		}
		return nil
	})
	return keep, it
}

func filterActor(it pub.Item, ff *Filters) (bool, pub.Item) {
	if ff == nil {
		return true, it
	}
	keep := true
	pub.OnActor(it, func(ob *pub.Actor) error {
		if !filterNaturalLanguageValues(ff.Names(), ob.Name, ob.PreferredUsername) {
			keep = false
			return nil
		}
		pub.OnObject(it, func(ob *pub.Object) error {
			keep = filterObjectNoName(ob, ff)
			return nil
		})
		return nil
	})
	return keep, it
}

func matchStringFilters(filters CompStrs, s string) bool {
	for _, f := range filters {
		if f.Operator == "!" && !matchStringFilter(f, s) {
			return false
		}
		if matchStringFilter(f, s) {
			return true
		}
	}
	return false
}

func matchStringFilter(filter CompStr, s string) bool {
	if filter.Operator == "~" {
		return strings.Contains(strings.ToLower(s), strings.ToLower(filter.Str))
	} else if filter.Operator == "!" {
		return !strings.Contains(strings.ToLower(s), strings.ToLower(filter.Str))
	}
	return strings.ToLower(s) == strings.ToLower(filter.Str)
}

func filterNaturalLanguageValues(filters CompStrs, valArr ...pub.NaturalLanguageValues) bool {
	keep := true
	if len(filters) > 0 {
		keep = false
	}
	for _, langValues := range valArr {
		for _, langValue := range langValues {
			if matchStringFilters(filters, langValue.Value) {
				keep = true
				break
			}
		}
	}
	return keep
}

func filterItems(filters CompStrs, items ...pub.Item) bool {
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
		if filterItem(filters, it) {
			return true
		}
	}
	return false
}

func filterAudience(filters CompStrs, colArr ...pub.ItemCollection) bool {
	if len(filters) == 0 {
		return true
	}
	allItems := make(pub.ItemCollection, 0)
	for _, items := range colArr {
		for _, it := range items {
			if it != nil && !allItems.Contains(it.GetLink()) {
				allItems = append(allItems, it)
			}
		}
	}
	return filterItems(filters, pub.ItemCollectionDeduplication(&allItems)...)
}

func filterItemCollections(filters CompStrs, colArr ...pub.Item) bool {
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

func hasAbsentFilter(filters CompStrs) bool {
	if len(filters) != 1 {
		return false
	}
	return filters[0].Str == AbsentIRI.Str
}

// filterAbsent is used when searching that the incoming items collection is empty
func filterAbsent(filters CompStrs, items ...pub.Item) bool {
	if filters[0].Str == AbsentIRI.Str {
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
			if it.GetLink() != pub.PublicNS { // FIXME(marius): this is kinda ugly
				return false
			}
		}
	}
	return true
}

func filterWithAbsent(filters CompStrs, items ...pub.Item) bool {
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

func filterItem(filters CompStrs, it pub.Item) bool {
	if len(filters) > 0 {
		if it == nil {
			return false
		}
		if c, ok := it.(pub.ItemCollection); ok {
			return filterItems(filters, c...)
		} else {
			return matchStringFilters(filters, it.GetLink().String())
		}
	}
	return false
}

func filterURLs(filters CompStrs, it pub.Item) bool {
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
		if ob.URL != nil {
			url = ob.URL.GetLink().String()
		}
	case *pub.Page:
		if ob.URL != nil {
			url = ob.URL.GetLink().String()
		}
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
		if filter.Operator == "~" {
			if strings.Contains(url, filter.Str) {
				keep = true
				break
			}
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
func iriPointsToCollection(iri pub.IRI) bool {
	if u, err := iri.URL(); err == nil {
		base := path.Base(u.Path)
		return !ValidCollection(h.CollectionType(base)) && base != "/"
	}
	return false
}

// ItemMatches
func (f *Filters) ItemMatches(it pub.Item) bool {
	if f == nil {
		return true
	}
	if it == nil {
		return false
	}
	iris := f.IRIs()
	// FIXME(marius): the Contains method returns true for the case where IRIs is empty, we don't want that
	if len(iris) > 0 && !filterItem(iris, it) {
		return false
	}
	types := f.Types()
	// FIXME(marius): this does not cover case insensitivity
	if len(types) > 0 && !types.Contains(it.GetType()) {
		return false
	}
	iri := f.GetLink()
	if len(iri) > 0 && iriPointsToCollection(iri) {
		if !it.GetLink().Contains(iri, false) {
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

// LoadCollectionFilters uses specific logic for adding elements to the filters when loading
// collections from the database.
func LoadCollectionFilters(r *http.Request, f *Filters) error {
	return LoadItemFilters(r, f)
}

// LoadItemFilters uses specific logic for adding elements to the filters when loading
// single items from the database.
func LoadItemFilters(r *http.Request, f *Filters) error {
	if len(f.Key) != 0 {
		for _, k := range f.Key {
			i := CompStr{Str: fmt.Sprintf("%s%s", f.IRI, k)}
			f.URL = append(f.URL, i)
		}
	}

	if auth, ok := auth.ActorContext(r.Context()); ok {
		f.Authenticated = &auth
		if f.Object != nil {
			f.Object.Authenticated = f.Authenticated
		}
		if f.Actor != nil {
			f.Actor.Authenticated = f.Authenticated
		}
		if f.Target != nil {
			f.Target.Authenticated = f.Authenticated
		}
	}

	return nil
}

// FilterIt
func FilterIt(it pub.Item, f s.Filterable) (pub.Item, error) {
	if it == nil {
		return it, nil
	}
	if ff, ok := f.(ItemMatcher); ok {
		if ff.ItemMatches(it) {
			return it, nil
		} else {
			return nil, nil
		}
	}
	if f1, ok := f.(s.Filterable); ok {
		if f1.GetLink().Equals(it.GetLink(), false) {
			return it, nil
		} else {
			return nil, nil
		}
	}
	if f1, ok := f.(s.FilterableItems); ok {
		iris := f1.IRIs()
		// FIXME(marius): the Contains method returns true for the case where IRIs is empty, we don't want that
		if len(iris) > 0 && !iris.Contains(it.GetLink()) {
			return nil, nil
		}
		types := f1.Types()
		// FIXME(marius): this does not cover case insensitivity
		if len(types) > 0 && !types.Contains(it.GetType()) {
			return nil, nil
		}
		return it, nil
	}
	return nil, errors.Errorf("Invalid filter %T", f)
}
