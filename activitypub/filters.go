package activitypub

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	h "github.com/go-ap/handlers"
	s "github.com/go-ap/storage"
	"github.com/mariusor/qstring"
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
		if c.Str == f.Str && c.Operator == f.Operator {
			return true
		}
	}
	return false
}

// String returns the hash as a string
func (h Hash) String() string {
	return string(h)
}

// Matches
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

	// ModType is an internally used collection, to store a list of actors the instance
	ModType = h.CollectionType("mods")
)

// TODO(marius): here we need a better separation between the collections which are exposed in the HTTP API
//   (activities,actors,objects) and the ones that are internal (blocked,ignored)
var (
	HiddenCollections = h.CollectionTypes{
		BlockedType,
		IgnoredType,
	}

	FedBOXCollections = h.CollectionTypes{
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
	baseURL       pub.IRI          `qstring:"-"`
	Name          CompStrs         `qstring:"name,omitempty"`
	Cont          CompStrs         `qstring:"content,omitempty"`
	Authenticated *pub.Actor       `qstring:"-"`
	To            *pub.Actor       `qstring:"-"`
	Author        *pub.Actor       `qstring:"-"`
	Parent        *pub.Actor       `qstring:"-"`
	IRI           pub.IRI          `qstring:"-"`
	Collection    h.CollectionType `qstring:"-"`
	URL           CompStrs         `qstring:"url,omitempty"`
	MedTypes      []pub.MimeType   `qstring:"mediaType,omitempty"`
	Aud           CompStrs         `qstring:"recipients,omitempty"`
	Gen           CompStrs         `qstring:"generator,omitempty"`
	Key           []Hash           `qstring:"-"`
	ItemKey       CompStrs         `qstring:"iri,omitempty"`
	Type          CompStrs         `qstring:"type,omitempty"`
	AttrTo        CompStrs         `qstring:"attributedTo,omitempty"`
	InReplTo      CompStrs         `qstring:"inReplyTo,omitempty"`
	OP            CompStrs         `qstring:"context,omitempty"`
	FollowedBy    []Hash           `qstring:"followedBy,omitempty"` // todo(marius): not really used
	OlderThan     time.Time        `qstring:"olderThan,omitempty"`
	NewerThan     time.Time        `qstring:"newerThan,omitempty"`
	Prev          Hash             `qstring:"before,omitempty"`
	Next          Hash             `qstring:"after,omitempty"`
	Object        *Filters         `qstring:"object,omitempty"`
	Actor         *Filters         `qstring:"actor,omitempty"`
	Target        *Filters         `qstring:"target,omitempty"`
	Tag           *Filters         `qstring:"tag,omitempty"`
	CurPage       uint             `qstring:"page,omitempty"`
	MaxItems      uint             `qstring:"maxItems,omitempty"`
	Req           *http.Request    `qstring:"-"`
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
		if f.Type == nil {
			f.Type = make(CompStrs, 0)
		}
		for _, t := range types {
			f.Type = append(f.Type, StringEquals(string(t)))
		}
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
func (f Filters) Types() CompStrs {
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

func (f *Filters) IsItemIRI() bool {
	u, err := f.GetLink().URL()
	if err != nil {
		return false
	}
	maybeID := h.CollectionType(path.Base(u.Path))
	maybeCol := h.CollectionType(path.Base(path.Dir(u.Path)))
	return !(FedBOXCollections.Contains(maybeID) || h.OnActor.Contains(maybeID) || h.OnObject.Contains(maybeID)) &&
		(FedBOXCollections.Contains(maybeCol) || h.OnActor.Contains(maybeCol) || h.OnObject.Contains(maybeCol))
}

// GetLink returns a list of IRIs to filter against
func (f *Filters) GetLink() pub.IRI {
	if f == nil {
		return ""
	}
	if f.IRI == "" {
		f.IRI = f.baseURL.AddPath(string(f.Collection))
	}
	iri := f.IRI
	if v, err := qstring.Marshal(f); err == nil && len(v) > 0 {
		iri = pub.IRI(fmt.Sprintf("%s?%s", iri, v.Encode()))
	}
	if f.Authenticated != nil && !f.Authenticated.GetLink().Equals(pub.PublicNS, false) {
		if u, err := iri.URL(); err == nil {
			u.User = url.User(f.Authenticated.ID.String())
			iri = pub.IRI(u.String())
		}
	}
	return iri
}

// Page
func (f Filters) Page() uint {
	return f.CurPage
}

// Before
func (f Filters) Before() Hash {
	return f.Prev
}

// After
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

func fullURL(u *url.URL) string {
	return fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, u.Path)
}

func baseURL(u *url.URL) string {
	return fmt.Sprintf("%s://%s", u.Scheme, u.Host)
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
	if f.Authenticated != nil && !f.Authenticated.GetLink().Equals(pub.PublicNS, false) {
		if user := StringEquals(f.Authenticated.GetLink().String()); !col.Contains(user) {
			col = append(col, user)
		}
	}
	public := StringEquals(pub.PublicNS.String())
	notPublic := StringDifferent(pub.PublicNS.String())
	if !col.Contains(public) && !col.Contains(notPublic) {
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

// Tags returns a list of filters to filter an object's tag collections against
func (f Filters) Tags() *Filters {
	return f.Tag
}

func filterObjectNoNameNoType(ob *pub.Object, ff *Filters) bool {
	if ff == nil {
		return true
	}
	if iris := ff.IRIs(); len(iris) > 0 {
		if !filterItem(iris, ob) {
			return false
		}
	}
	if !filterNaturalLanguageValues(ff.Content(), ob.Content, ob.Summary) {
		return false
	}
	if !filterWithAbsent(ff.Generator(), ob.Generator) {
		return false
	}
	if !filterURLs(ff.URLs(), ob) {
		return false
	}
	if !filterWithAbsent(ff.Context(), ob.Context, ob.InReplyTo) {
		return false
	}
	if !filterWithAbsent(ff.InReplyTo(), ob.InReplyTo) {
		return false
	}
	if !filterWithAbsent(ff.AttributedTo(), ob.AttributedTo) {
		return false
	}
	if !filterMediaTypes(ff.MediaTypes(), ob.MediaType) {
		return false
	}
	if !ff.Tags().ItemsMatch(ob.Tag) {
		return false
	}
	return true
}

func filterTypes(filters CompStrs, types ...pub.ActivityVocabularyType) bool {
	match := len(filters) == 0
	for _, filter := range filters {
		if filter.Operator == "!" {
			match = !match
		}
		for _, typ := range types {
			m := matchStringFilter(filter, string(typ))
			if filter.Operator == "!" {
				match = match && m
			} else {
				match = match || m
			}
		}
	}
	return match
}

func filterTombstone(it pub.Item, ff *Filters) (bool, pub.Item) {
	keep := true
	pub.OnTombstone(it, func(t *pub.Tombstone) error {
		keep = filterTypes(ff.Types(), t.FormerType, t.Type)
		return nil
	})
	if !keep {
		return keep, it
	}
	if len(ff.Name) > 0 {
		return false, it
	}
	pub.OnObject(it, func(ob *pub.Object) error {
		keep = filterObjectNoNameNoType(ob, ff)
		return nil
	})
	return keep, it
}

func filterLink(it pub.Item, ff *Filters) (bool, pub.Item) {
	if ff == nil {
		return true, it
	}
	if !it.IsLink() {
		return false, it
	}
	keep := true
	pub.OnLink(it, func(l *pub.Link) error {
		if keep = filterNaturalLanguageValues(ff.Names(), l.Name); !keep {
			return nil
		}
		if iris := ff.IRIs(); len(iris) > 0 {
			if !filterItem(iris, l) {
				keep = false
			}
		}
		if !filterURLs(ff.URLs(), l) {
			keep = false
		}
		if !filterMediaTypes(ff.MediaTypes(), l.MediaType) {
			keep = false
		}
		return nil
	})
	if !keep {
		return keep, it
	}
	return filterTypes(ff.Types(), it.GetType()), it
}

func filterObject(it pub.Item, ff *Filters) (bool, pub.Item) {
	if ff == nil {
		return true, it
	}
	if !it.IsObject() {
		return false, it
	}
	keep := true
	pub.OnObject(it, func(ob *pub.Object) error {
		if keep = filterNaturalLanguageValues(ff.Names(), ob.Name); !keep {
			return nil
		}
		keep = filterObjectNoNameNoType(ob, ff)
		return nil
	})
	if !keep {
		return keep, it
	}
	return filterTypes(ff.Types(), it.GetType()), it
}

// NOTE(marius): this is being called even if it is an IntransitiveActivity
//  and probably will crash when accessing act.Object
func filterActivity(it pub.Item, ff *Filters) (bool, pub.Item) {
	if ff == nil {
		return true, it
	}
	keep := true
	pub.OnActivity(it, func(act *pub.Activity) error {
		if keep, _ = filterObject(act, ff); !keep {
			return nil
		}
		if keep = ff.Actor.ItemsMatch(act.Actor); !keep {
			return nil
		}
		if keep = ff.Object.ItemsMatch(act.Object); !keep {
			return nil
		}
		keep = ff.Target.ItemsMatch(act.Target)
		return nil
	})
	if !keep {
		return keep, it
	}
	return filterTypes(ff.Types(), it.GetType()), it
}

func filterActor(it pub.Item, ff *Filters) (bool, pub.Item) {
	if ff == nil {
		return true, it
	}
	keep := true
	pub.OnActor(it, func(ob *pub.Actor) error {
		keep = filterNaturalLanguageValues(ff.Names(), ob.Name, ob.PreferredUsername)
		return nil
	})
	if !keep {
		return keep, it
	}
	pub.OnObject(it, func(ob *pub.Object) error {
		keep = filterObjectNoNameNoType(ob, ff)
		return nil
	})
	if !keep {
		return keep, it
	}
	return filterTypes(ff.Types(), it.GetType()), it
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

func matchNaturalLanguageValues(filters CompStrs, s pub.NaturalLanguageValues) bool {
	var match bool
	for _, f := range filters {
		match = match || matchLangRefs(f, s...)
	}
	return match
}

func matchLangRefs(filter CompStr, refs ...pub.LangRefValue) bool {
	var match bool
	if filter.Operator == "!" {
		match = !match
	}
	for _, ref := range refs {
		m := matchStringFilter(filter, ref.Value.String())
		if filter.Operator == "!" {
			match = match && m
		} else {
			match = match || m
		}
	}
	return match
}

func matchStringFilter(filter CompStr, s string) bool {
	if filter.Operator == "~" {
		return strings.Contains(strings.ToLower(s), strings.ToLower(filter.Str))
	} else if filter.Operator == "!" {
		if filter.Str == "" || filter.Str == pub.NilIRI.String() {
			return len(s) > 0
		}
		return !strings.Contains(strings.ToLower(s), strings.ToLower(filter.Str))
	}
	return strings.ToLower(s) == strings.ToLower(filter.Str)
}

func filterNaturalLanguageValues(filters CompStrs, valArr ...pub.NaturalLanguageValues) bool {
	keep := len(filters) == 0
	for _, langValues := range valArr {
		if matchNaturalLanguageValues(filters, langValues) {
			keep = true
			break
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
		if pub.IsNil(it) {
			continue
		}
		if filterItem(filters, it) {
			return true
		}
	}
	return false
}

func FilterAudience(filters CompStrs, colArr ...pub.ItemCollection) bool {
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
	keep := (len(filters) == 1 && filters[0].String() == pub.PublicNS.String()) && len(allItems) == 0
	for _, f := range filters {
		for _, it := range allItems {
			s := it.GetLink().String()
			if f.Operator == "!" && !matchStringFilter(f, s) {
				keep = false
				break
			}
			if matchStringFilter(f, s) {
				keep = true
			}
		}
	}
	return keep
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
			if pub.IsNil(it) {
				continue
			}
			if it.IsCollection() {
				result := false
				pub.OnCollectionIntf(it, func(c pub.CollectionInterface) error {
					for _, it := range c.Collection() {
						if pub.IsNil(it) {
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
		if pub.IsNil(it) {
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
	if pub.IsNil(it) {
		return false
	}
	var url pub.IRI
	pub.OnObject(it, func(o *pub.Object) error {
		if o.URL != nil {
			url = o.URL.GetLink()
		}
		return nil
	})
	for _, filter := range filters {
		filterIRI := pub.IRI(filter.Str)
		if filter.Operator == "~" {
			if url.Contains(filterIRI, false) {
				keep = true
				break
			}
		} else if filter.Operator == "!" {
			if !url.Equals(filterIRI, false) {
				keep = true
				break
			}
		} else {
			if url.Equals(filterIRI, false) {
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
	ItemsMatch(it ...pub.Item) bool
}

func (f Filters) FilterCollection(col pub.ItemCollection) (pub.ItemCollection, int) {
	if len(col) == 0 {
		return col, 0
	}
	new := make(pub.ItemCollection, len(col))
	for _, it := range col {
		if f.ItemsMatch(it) {
			new = append(new, it)
		}
	}
	col = new
	return nil, 0
}

// ItemsMatch
func (f *Filters) ItemsMatch(col ...pub.Item) bool {
	if f == nil {
		return true
	}
	if col == nil {
		return false
	}
	var valid bool
	mustBeActivity := f.Object != nil || f.Actor != nil || f.Target != nil
	mustBeObject := f.Tag != nil || f.AttrTo != nil || f.InReplTo != nil || f.Author != nil
	for _, it := range col {
		if pub.IsNil(it) {
			continue
		}
		var loopValid bool
		if it.IsCollection() {
			if mustBeActivity || mustBeObject {
				return false
			}
			pub.OnCollectionIntf(it, func(col pub.CollectionInterface) error {
				loopValid = f.ItemsMatch(col.Collection()...)
				return nil
			})
		} else if it.IsObject() {
			typ := it.GetType()
			if pub.ActivityTypes.Contains(typ) || pub.IntransitiveActivityTypes.Contains(typ) {
				loopValid, _ = filterActivity(it, f)
			} else if pub.ActorTypes.Contains(typ) {
				if mustBeActivity {
					return false
				}
				loopValid, _ = filterActor(it, f)
			} else if typ == pub.TombstoneType {
				if mustBeActivity {
					return false
				}
				loopValid, _ = filterTombstone(it, f)
			} else {
				if mustBeActivity {
					return false
				}
				loopValid, _ = filterObject(it, f)
			}
		} else if it.IsLink() {
			if mustBeActivity || mustBeObject {
				return false
			}
			loopValid, _ = filterLink(it, f)
		}
		valid = valid || loopValid
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
	if pub.IsNil(it) {
		return it, nil
	}
	if ff, ok := f.(ItemMatcher); ok {
		if ff.ItemsMatch(it) {
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

func FiltersOnActivityObject(f s.Filterable) bool {
	if ff, ok := f.(*Filters); ok {
		return ff.Object != nil
	}
	return false
}

func FiltersOnActivityActor(f s.Filterable) bool {
	if ff, ok := f.(*Filters); ok {
		return ff.Actor != nil
	}
	return false
}

// CacheKey generates a unique pub.IRI hash based on its authenticated user and other parameters
func CacheKey(f *Filters) pub.IRI {
	var iri pub.IRI

	if q, err := qstring.Marshal(f); err == nil && len(q) > 0 {
		iri = pub.IRI(fmt.Sprintf("%s?%s", f.GetLink(), q.Encode()))
	} else {
		iri = f.GetLink()
	}
	u, _ := iri.URL()
	if auth := f.Authenticated; auth != nil && !auth.ID.Equals(pub.PublicNS, true) {
		u.User = url.User(path.Base(f.Authenticated.ID.String()))
	}
	return pub.IRI(u.String())
}

func FiltersFromIRI (i pub.IRI) (*Filters, error) {
	f := FiltersNew()
	u, _ := i.URL()
	if f.baseURL == "" {
		f.baseURL = pub.IRI(baseURL(u))
	}
	if u.User != nil {
		if us, err := url.Parse(u.User.Username()); err == nil {
			id := pub.IRI(us.String())
			if id != pub.PublicNS {
				f.Authenticated = &pub.Actor{ID: id}
			}
		}
	}
	if err := qstring.Unmarshal(u.Query(), f); err != nil {
		return f, err
	}
	if len(f.IRI) == 0 {
		f.IRI = pub.IRI(fullURL(u))
	}
	if f.Collection == "" {
		req := new(http.Request)
		req.URL = u
		f.Collection = h.Typer.Type(req)
	}

	if f.MaxItems > MaxItems {
		f.MaxItems = MaxItems
	}

	return f, nil
}

// FromRequest loads the filters we use for generating storage queries from the HTTP request
func FromRequest(r *http.Request, baseUrl string) (*Filters, error) {
	f := FiltersNew()
	f.Req = r

	var u *url.URL
	if baseUrl != "" {
		f.baseURL = pub.IRI(baseUrl)
		u, _ = f.baseURL.URL()
		u.Path = r.URL.Path
	} else {
		f.baseURL = pub.IRI(baseURL(r.URL))
		u = r.URL
	}
	if len(f.IRI) == 0 {
		f.IRI = pub.IRI(fullURL(u))
	}
	if err := qstring.Unmarshal(r.URL.Query(), f); err != nil {
		return f, err
	}
	f.Collection = h.Typer.Type(r)

	if f.MaxItems > MaxItems {
		f.MaxItems = MaxItems
	}

	return f, nil
}
