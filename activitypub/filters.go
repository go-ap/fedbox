package activitypub

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"time"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	"github.com/go-ap/processing"
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
func (h Hash) Matches(i vocab.IRI) bool {
	return path.Base(i.String()) == string(h)
}

const (
	// ActorsType is a constant that represents the URL path for the local actors collection.
	// It is used as the parent for all To IDs
	ActorsType = vocab.CollectionPath("actors")
	// ActivitiesType is a constant that represents the URL path for the local activities collection
	// It is used as the parent for all Activity IDs
	ActivitiesType = vocab.CollectionPath("activities")
	// ObjectsType is a constant that represents the URL path for the local objects collection
	// It is used as the parent for all non To, non Activity Object IDs
	ObjectsType = vocab.CollectionPath("objects")

	// BlockedType is an internally used collection, to store a list of actors the actor has blocked
	BlockedType = vocab.CollectionPath("blocked")

	// IgnoredType is an internally used collection, to store a list of actors the actor has ignored
	IgnoredType = vocab.CollectionPath("ignored")
)

// TODO(marius): here we need a better separation between the collections which are exposed in the HTTP API
//
//	(activities,actors,objects) and the ones that are internal (blocked,ignored)
var (
	HiddenCollections = vocab.CollectionPaths{
		BlockedType,
		IgnoredType,
	}

	FedBOXCollections = vocab.CollectionPaths{
		ActivitiesType,
		ActorsType,
		ObjectsType,
		BlockedType,
		IgnoredType,
	}

	validActivityCollection = vocab.CollectionPaths{
		ActivitiesType,
	}

	validObjectCollection = vocab.CollectionPaths{
		ActorsType,
		ObjectsType,
	}
)

func getValidActivityCollection(typ vocab.CollectionPath) vocab.CollectionPath {
	for _, t := range validActivityCollection {
		if strings.ToLower(string(typ)) == string(t) {
			return t
		}
	}
	return vocab.Unknown
}

func getValidObjectCollection(typ vocab.CollectionPath) vocab.CollectionPath {
	for _, t := range validObjectCollection {
		if strings.ToLower(string(typ)) == string(t) {
			return t
		}
	}
	return vocab.Unknown
}

// ValidCollection shows if the current ActivityPub end-point type is a valid collection
func ValidCollection(typ vocab.CollectionPath) bool {
	return ValidActivityCollection(typ) || ValidObjectCollection(typ)
}

// ValidActivityCollection shows if the current ActivityPub end-point type is a valid collection for handling Activities
func ValidActivityCollection(typ vocab.CollectionPath) bool {
	return getValidActivityCollection(typ) != vocab.Unknown || vocab.ValidActivityCollection(typ)
}

// ValidObjectCollection shows if the current ActivityPub end-point type is a valid collection for handling Objects
func ValidObjectCollection(typ vocab.CollectionPath) bool {
	return getValidObjectCollection(typ) != vocab.Unknown || vocab.ValidObjectCollection(typ)
}

// Filters
// TODO(marius) we can make some small changes so it's not necessary to export this struct
type Filters struct {
	baseURL       vocab.IRI            `qstring:"-"`
	Name          CompStrs             `qstring:"name,omitempty"`
	Cont          CompStrs             `qstring:"content,omitempty"`
	Authenticated *vocab.Actor         `qstring:"-"`
	To            *vocab.Actor         `qstring:"-"`
	Author        *vocab.Actor         `qstring:"-"`
	Parent        *vocab.Actor         `qstring:"-"`
	IRI           vocab.IRI            `qstring:"-"`
	Collection    vocab.CollectionPath `qstring:"-"`
	URL           CompStrs             `qstring:"url,omitempty"`
	MedTypes      []vocab.MimeType     `qstring:"mediaType,omitempty"`
	Aud           CompStrs             `qstring:"recipients,omitempty"`
	Gen           CompStrs             `qstring:"generator,omitempty"`
	Key           []Hash               `qstring:"-"`
	ItemKey       CompStrs             `qstring:"iri,omitempty"`
	Type          CompStrs             `qstring:"type,omitempty"`
	AttrTo        CompStrs             `qstring:"attributedTo,omitempty"`
	InReplTo      CompStrs             `qstring:"inReplyTo,omitempty"`
	OP            CompStrs             `qstring:"context,omitempty"`
	FollowedBy    []Hash               `qstring:"followedBy,omitempty"` // todo(marius): not really used
	OlderThan     time.Time            `qstring:"olderThan,omitempty"`
	NewerThan     time.Time            `qstring:"newerThan,omitempty"`
	Prev          Hash                 `qstring:"before,omitempty"`
	Next          Hash                 `qstring:"after,omitempty"`
	Object        *Filters             `qstring:"object,omitempty"`
	Actor         *Filters             `qstring:"actor,omitempty"`
	Target        *Filters             `qstring:"target,omitempty"`
	Tag           *Filters             `qstring:"tag,omitempty"`
	CurPage       uint                 `qstring:"page,omitempty"`
	MaxItems      uint                 `qstring:"maxItems,omitempty"`
	Req           *http.Request        `qstring:"-"`
}

func ItemKey(keys ...string) FilterFn {
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

func IRI(i vocab.IRI) FilterFn {
	return func(f *Filters) error {
		f.IRI = i
		return nil
	}
}

func Name(names ...string) FilterFn {
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

func Type(types ...vocab.ActivityVocabularyType) FilterFn {
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

func BaseIRI(iri vocab.IRI, col vocab.CollectionPath) FilterFn {
	return func(f *Filters) error {
		f.baseURL = iri
		f.Collection = col
		return nil
	}
}

type FilterFn func(f *Filters) error

func FiltersNew(filters ...FilterFn) *Filters {
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
		if vocab.ValidObjectCollection(f.Collection) {
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
	maybeID := vocab.CollectionPath(path.Base(u.Path))
	maybeCol := vocab.CollectionPath(path.Base(path.Dir(u.Path)))
	return !(FedBOXCollections.Contains(maybeID) || vocab.OfActor.Contains(maybeID) || vocab.OfObject.Contains(maybeID)) &&
		(FedBOXCollections.Contains(maybeCol) || vocab.OfActor.Contains(maybeCol) || vocab.OfObject.Contains(maybeCol))
}

// GetLink returns a list of IRIs to filter against
func (f *Filters) GetLink() vocab.IRI {
	if f == nil {
		return ""
	}
	if f.IRI == "" {
		f.IRI = f.baseURL.AddPath(string(f.Collection))
	}
	iri := f.IRI
	if v, err := qstring.Marshal(f); err == nil && len(v) > 0 {
		iri = vocab.IRI(fmt.Sprintf("%s?%s", iri, v.Encode()))
	}
	if f.Authenticated != nil && !f.Authenticated.GetLink().Equals(vocab.PublicNS, false) {
		if u, err := iri.URL(); err == nil {
			u.User = url.User(f.Authenticated.ID.String())
			iri = vocab.IRI(u.String())
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
	return errors.Errorf("%s not found", s)
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
	if f.Authenticated != nil && !f.Authenticated.GetLink().Equals(vocab.PublicNS, false) {
		if user := StringEquals(f.Authenticated.GetLink().String()); !col.Contains(user) {
			col = append(col, user)
		}
	}
	public := StringEquals(vocab.PublicNS.String())
	notPublic := StringDifferent(vocab.PublicNS.String())
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

func (f Filters) MediaTypes() []vocab.MimeType {
	return f.MedTypes
}

func (f Filters) URLs() CompStrs {
	return f.URL
}

func (f Filters) Generator() CompStrs {
	return f.Gen
}

func (f Filters) Actors() vocab.IRIs {
	ret := make(vocab.IRIs, 0)
	if f.Actor == nil {
		return nil
	}
	for _, k := range f.Actor.Key {
		// TODO(marius): This piece of logic should be moved to loading the filters
		f.Collection = ActorsType
		iri := vocab.IRI(IRIf(f, k.String()))
		if !ret.Contains(iri) {
			ret = append(ret, iri)
		}
	}
	return ret
}

func (f Filters) Objects() vocab.IRIs {
	ret := make(vocab.IRIs, 0)
	if f.Object == nil {
		return nil
	}
	for _, k := range f.Object.Key {
		// TODO(marius): This piece of logic should be moved to loading the filters
		f.Collection = ObjectsType
		iri := vocab.IRI(IRIf(f, k.String()))
		if !ret.Contains(iri) {
			ret = append(ret, iri)
		}
	}
	return ret
}

func (f Filters) Targets() vocab.IRIs {
	ret := make(vocab.IRIs, 0)
	if f.Target == nil {
		return nil
	}
	for _, k := range f.Target.Key {
		// TODO(marius): This piece of logic should be moved to loading the filters
		var iris vocab.IRIs
		if u, ok := validURL(k.String()); ok {
			iris = vocab.IRIs{vocab.IRI(u.String())}
		} else {
			// FIXME(marius): we don't really know which type this is
			iris = vocab.IRIs{
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

func filterObjectNoNameNoType(ob *vocab.Object, ff *Filters) bool {
	if ff == nil {
		return true
	}
	if iris := ff.IRIs(); len(iris) > 0 {
		if !filterItem(iris, ob) {
			return false
		}
	}
	if !filterAudience(ff.Audience(), ob.Recipients(), vocab.ItemCollection{ob.AttributedTo}) {
		return false
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

func filterTypes(filters CompStrs, types ...vocab.ActivityVocabularyType) bool {
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

func filterTombstone(it vocab.Item, ff *Filters) (bool, vocab.Item) {
	keep := true
	vocab.OnTombstone(it, func(t *vocab.Tombstone) error {
		keep = filterTypes(ff.Types(), t.FormerType, t.Type)
		return nil
	})
	if !keep {
		return keep, it
	}
	if len(ff.Name)+len(ff.Content()) > 0 {
		return false, it
	}
	vocab.OnObject(it, func(ob *vocab.Object) error {
		keep = filterObjectNoNameNoType(ob, ff)
		return nil
	})
	return keep, it
}

func filterIRI(it vocab.IRI, ff *Filters) (bool, vocab.IRI) {
	keep := true
	if iris := ff.IRIs(); len(iris) > 0 {
		if !filterItem(iris, it) {
			keep = false
		}
	}
	if !filterURLs(ff.URLs(), it) {
		keep = false
	}
	return keep, it
}

func filterLink(it vocab.Item, ff *Filters) (bool, vocab.Item) {
	if ff == nil {
		return true, it
	}
	if !it.IsLink() {
		return false, it
	}
	if vocab.IsIRI(it) {
		return filterIRI(it.GetLink(), ff)
	}
	keep := true
	vocab.OnLink(it, func(l *vocab.Link) error {
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

func filterObject(it vocab.Item, ff *Filters) (bool, vocab.Item) {
	if ff == nil {
		return true, it
	}
	if !it.IsObject() {
		return false, it
	}
	keep := true
	vocab.OnObject(it, func(ob *vocab.Object) error {
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

func filterIntransitiveActivity(it vocab.Item, ff *Filters) (bool, vocab.Item) {
	if ff == nil {
		return true, it
	}
	keep := true
	vocab.OnIntransitiveActivity(it, func(act *vocab.IntransitiveActivity) error {
		if keep, _ = filterObject(act, ff); !keep {
			return nil
		}
		if keep = ff.Actor.ItemsMatch(act.Actor); !keep {
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

func filterActivity(it vocab.Item, ff *Filters) (bool, vocab.Item) {
	if ff == nil {
		return true, it
	}
	keep := true
	vocab.OnActivity(it, func(act *vocab.Activity) error {
		if keep, _ = filterIntransitiveActivity(act, ff); !keep {
			return nil
		}
		if keep = ff.Object.ItemsMatch(act.Object); !keep {
			return nil
		}
		return nil
	})
	if !keep {
		return keep, it
	}
	return filterTypes(ff.Types(), it.GetType()), it
}

func filterActor(it vocab.Item, ff *Filters) (bool, vocab.Item) {
	if ff == nil {
		return true, it
	}
	keep := true
	vocab.OnActor(it, func(ob *vocab.Actor) error {
		keep = filterNaturalLanguageValues(ff.Names(), ob.Name, ob.PreferredUsername)
		return nil
	})
	if !keep {
		return keep, it
	}
	vocab.OnObject(it, func(ob *vocab.Object) error {
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

func matchNaturalLanguageValues(filters CompStrs, s vocab.NaturalLanguageValues) bool {
	var match bool
	for _, f := range filters {
		match = match || matchLangRefs(f, s...)
	}
	return match
}

func matchLangRefs(filter CompStr, refs ...vocab.LangRefValue) bool {
	var match bool
	if filter.Operator == "!" {
		match = !match
		if len(refs) == 0 {
			return false
		}
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
		if filter.Str == "" || filter.Str == vocab.NilIRI.String() {
			return len(s) > 0
		}
		return !strings.Contains(strings.ToLower(s), strings.ToLower(filter.Str))
	}
	return strings.ToLower(s) == strings.ToLower(filter.Str)
}

func filterNaturalLanguageValues(filters CompStrs, valArr ...vocab.NaturalLanguageValues) bool {
	keep := len(filters) == 0
	for _, langValues := range valArr {
		if matchNaturalLanguageValues(filters, langValues) {
			keep = true
			break
		}
	}
	return keep
}

func filterItems(filters CompStrs, items ...vocab.Item) bool {
	if len(filters) == 0 {
		return true
	}
	if hasAbsentFilter(filters) && filterAbsent(filters, items...) {
		return true
	}
	for _, it := range items {
		if vocab.IsNil(it) {
			continue
		}
		if filterItem(filters, it) {
			return true
		}
	}
	return false
}

func filterAudience(filters CompStrs, allAudienceCollection ...vocab.ItemCollection) bool {
	if len(filters) == 0 {
		return true
	}
	audience := make(vocab.ItemCollection, 0)
	for _, items := range allAudienceCollection {
		for _, it := range items {
			if it != nil && !audience.Contains(it.GetLink()) {
				audience = append(audience, it)
			}
		}
	}
	if len(audience) == 0 || audience.Contains(vocab.PublicNS) {
		// Object is public
		return true
	}
	keep := len(filters) == 1 && filters[0].String() == vocab.PublicNS.String()
	for _, f := range filters {
		for _, it := range audience {
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

func filterItemCollections(filters CompStrs, colArr ...vocab.Item) bool {
	if len(filters) == 0 {
		return true
	}

	allItems := make(vocab.ItemCollection, 0)
	for _, col := range colArr {
		if col == nil {
			continue
		}
		if col.IsCollection() {
			vocab.OnCollectionIntf(col, func(c vocab.CollectionInterface) error {
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
	vocab.ItemCollectionDeduplication(&allItems)
	return filterItems(filters, allItems...)
}

func hasAbsentFilter(filters CompStrs) bool {
	if len(filters) != 1 {
		return false
	}
	return filters[0].Str == AbsentIRI.Str
}

// filterAbsent is used when searching that the incoming items collection is empty
func filterAbsent(filters CompStrs, items ...vocab.Item) bool {
	if filters[0].Str == AbsentIRI.Str {
		if len(items) == 0 {
			return true
		}
		for _, it := range items {
			if vocab.IsNil(it) {
				continue
			}
			if it.IsCollection() {
				result := false
				vocab.OnCollectionIntf(it, func(c vocab.CollectionInterface) error {
					for _, it := range c.Collection() {
						if vocab.IsNil(it) {
							continue
						}
						if it != nil && it.GetLink() == vocab.PublicNS { // FIXME(marius): this is kinda ugly
							result = true
							return nil
						}
					}
					return nil
				})
				return result
			}
			if it.GetLink() != vocab.PublicNS { // FIXME(marius): this is kinda ugly
				return false
			}
		}
	}
	return true
}

func filterWithAbsent(filters CompStrs, items ...vocab.Item) bool {
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

func filterItem(filters CompStrs, it vocab.Item) bool {
	if len(filters) > 0 {
		if vocab.IsNil(it) {
			return false
		}
		if c, ok := it.(vocab.ItemCollection); ok {
			return filterItems(filters, c...)
		} else {
			return matchStringFilters(filters, it.GetLink().String())
		}
	}
	return false
}

func filterURLs(filters CompStrs, it vocab.Item) bool {
	if len(filters) == 0 {
		return true
	}
	keep := false
	if vocab.IsNil(it) {
		return false
	}
	var url vocab.IRI
	vocab.OnObject(it, func(o *vocab.Object) error {
		if o.URL != nil {
			url = o.URL.GetLink()
		}
		return nil
	})
	for _, filter := range filters {
		filterIRI := vocab.IRI(filter.Str)
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

func filterMediaTypes(medTypes []vocab.MimeType, typ vocab.MimeType) bool {
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
	FilterCollection(col vocab.ItemCollection) (vocab.ItemCollection, int)
}

type ItemMatcher interface {
	ItemsMatch(it ...vocab.Item) bool
}

func (f Filters) FilterCollection(col vocab.ItemCollection) (vocab.ItemCollection, int) {
	if len(col) == 0 {
		return col, 0
	}
	new := make(vocab.ItemCollection, len(col))
	for _, it := range col {
		if f.ItemsMatch(it) {
			new = append(new, it)
		}
	}
	col = new
	return nil, 0
}

// ItemsMatch
func (f *Filters) ItemsMatch(col ...vocab.Item) bool {
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
		if vocab.IsNil(it) {
			continue
		}
		var loopValid bool
		if it.IsCollection() {
			if mustBeActivity || mustBeObject {
				return false
			}
			vocab.OnCollectionIntf(it, func(col vocab.CollectionInterface) error {
				loopValid = f.ItemsMatch(col.Collection()...)
				return nil
			})
		} else if it.IsObject() {
			typ := it.GetType()
			if vocab.IntransitiveActivityTypes.Contains(typ) {
				loopValid, _ = filterIntransitiveActivity(it, f)
			} else if vocab.ActivityTypes.Contains(typ) {
				loopValid, _ = filterActivity(it, f)
			} else if vocab.ActorTypes.Contains(typ) {
				if mustBeActivity {
					return false
				}
				loopValid, _ = filterActor(it, f)
			} else if typ == vocab.TombstoneType {
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
func LoadCollectionFilters(f *Filters, auth *vocab.Actor) error {
	return LoadItemFilters(f, auth)
}

// LoadItemFilters uses specific logic for adding elements to the filters when loading
// single items from the database.
func LoadItemFilters(f *Filters, auth *vocab.Actor) error {
	if len(f.Key) != 0 {
		for _, k := range f.Key {
			i := CompStr{Str: fmt.Sprintf("%s%s", f.IRI, k)}
			f.URL = append(f.URL, i)
		}
	}

	if auth != nil && auth.ID != vocab.PublicNS {
		f.Authenticated = auth
		if f.Object != nil {
			f.Object.Authenticated = f.Authenticated
		}
		if f.Actor != nil {
			f.Actor.Authenticated = f.Authenticated
		}
		if f.Target != nil {
			f.Target.Authenticated = f.Authenticated
		}
		if f.Tag != nil {
			f.Tag.Authenticated = f.Authenticated
		}
	}
	return nil
}

func ValidCollectionCount(col vocab.ItemCollection, f processing.Filterable) bool {
	ff, ok := f.(*Filters)
	if !ok {
		return true
	}
	max := MaxItems
	if ff.MaxItems > 0 {
		max = ff.MaxItems
	}
	return uint(len(col)) < max
}

// FilterIt
func FilterIt(it vocab.Item, f processing.Filterable) (vocab.Item, error) {
	if vocab.IsNil(it) {
		return it, nil
	}
	if ff, ok := f.(ItemMatcher); ok {
		if ff.ItemsMatch(it) {
			return it, nil
		} else {
			return nil, nil
		}
	}
	if f1, ok := f.(processing.Filterable); ok {
		if f1.GetLink().Equals(it.GetLink(), false) {
			return it, nil
		} else {
			return nil, nil
		}
	}
	if f1, ok := f.(processing.FilterableItems); ok {
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

func FiltersOnActivityObject(f processing.Filterable) (bool, processing.Filterable) {
	if ff, ok := f.(*Filters); ok {
		return ff.Object != nil, ff.Object
	}
	return false, nil
}

func FiltersOnActivityActor(f processing.Filterable) (bool, processing.Filterable) {
	if ff, ok := f.(*Filters); ok {
		return ff.Actor != nil, ff.Actor
	}
	return false, nil
}

func FiltersOnActivityTarget(f processing.Filterable) (bool, processing.Filterable) {
	if ff, ok := f.(*Filters); ok {
		return ff.Target != nil, ff.Target
	}
	return false, nil
}

// CacheKey generates a unique vocab.IRI hash based on its authenticated user and other parameters
func CacheKey(f *Filters) vocab.IRI {
	var iri vocab.IRI

	if q, err := qstring.Marshal(f); err == nil && len(q) > 0 {
		iri = vocab.IRI(fmt.Sprintf("%s?%s", f.GetLink(), q.Encode()))
	} else {
		iri = f.GetLink()
	}
	u, _ := iri.URL()
	if auth := f.Authenticated; auth != nil && !auth.ID.Equals(vocab.PublicNS, true) {
		u.User = url.User(path.Base(f.Authenticated.ID.String()))
	}
	return vocab.IRI(u.String())
}

func FiltersFromIRI(i vocab.IRI) (*Filters, error) {
	f := FiltersNew()
	u, _ := i.URL()
	if f.baseURL == "" {
		f.baseURL = vocab.IRI(baseURL(u))
	}
	if u.User != nil {
		if us, err := url.Parse(u.User.Username()); err == nil {
			id := vocab.IRI(us.String())
			if id != vocab.PublicNS {
				f.Authenticated = &vocab.Actor{ID: id}
			}
		}
	}
	if err := qstring.Unmarshal(u.Query(), f); err != nil {
		return f, err
	}
	if len(f.IRI) == 0 {
		f.IRI = vocab.IRI(fullURL(u))
	}
	if f.Collection == "" {
		req := new(http.Request)
		req.URL = u
		f.Collection = processing.Typer.Type(req)
	}
	if f.Object != nil {
		f.Object.Authenticated = f.Authenticated
	}
	if f.Actor != nil {
		f.Actor.Authenticated = f.Authenticated
	}
	if f.Target != nil {
		f.Target.Authenticated = f.Authenticated
	}
	if f.Tag != nil {
		f.Tag.Authenticated = f.Authenticated
	}

	if f.MaxItems > MaxItems {
		f.MaxItems = MaxItems
	}

	return f, nil
}

// FromRequest loads the filters we use for generating storage queries from the HTTP request
func FromRequest(r *http.Request, baseUrl string) *Filters {
	f := FiltersNew()
	f.Req = r

	var u *url.URL
	if baseUrl != "" {
		f.baseURL = vocab.IRI(baseUrl)
		u, _ = f.baseURL.URL()
		u.Path = filepath.Clean(r.URL.Path)
	} else {
		f.baseURL = vocab.IRI(baseURL(r.URL))
		u = r.URL
	}
	if len(f.IRI) == 0 {
		f.IRI = vocab.IRI(fullURL(u))
	}
	f.Collection = processing.Typer.Type(r)
	qstring.Unmarshal(r.URL.Query(), f)

	if f.MaxItems > MaxItems {
		f.MaxItems = MaxItems
	}

	return f
}
