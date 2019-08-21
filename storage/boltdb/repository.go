package boltdb

import (
	"bytes"
	"fmt"
	"github.com/boltdb/bolt"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/handlers"
	"github.com/go-ap/jsonld"
	s "github.com/go-ap/storage"
	"github.com/sirupsen/logrus"
	"net/url"
	"path"
	"strings"
	"time"
)

type repo struct {
	d       *bolt.DB
	baseURL string
	root    []byte
	path    string
	logFn   loggerFn
	errFn   loggerFn
}

type loggerFn func(logrus.Fields, string, ...interface{})

const (
	bucketActors     = "actors"
	bucketActivities = "activities"
	bucketObjects    = "objects"
)

// Config
type Config struct {
	Path       string
	BucketName string
	LogFn      loggerFn
	ErrFn      loggerFn
}

// New returns a new repo repository
func New(c Config, baseURL string) *repo {
	b := repo{
		root:    []byte(c.BucketName),
		path:    c.Path,
		baseURL: baseURL,
		logFn:   func(logrus.Fields, string, ...interface{}) {},
		errFn:   func(logrus.Fields, string, ...interface{}) {},
	}
	if c.ErrFn != nil {
		b.errFn = c.ErrFn
	}
	if c.LogFn != nil {
		b.logFn = c.LogFn
	}
	return &b
}

func loadFromBucket(db *bolt.DB, root []byte, f s.Filterable) (as.ItemCollection, uint, error) {
	col := make(as.ItemCollection, 0)

	err := db.View(func(tx *bolt.Tx) error {
		rb := tx.Bucket(root)
		if rb == nil {
			return errors.Errorf("Invalid bucket %s", root)
		}

		var path string
		iri := f.GetLink()
		if iri != "" {
			// This is the case where the Filter points to a single AP Object IRI
			// TODO(marius): Ideally this should support the case where we use the IRI to point to a bucket path
			//     and on top of that apply the other filters
			url, err := iri.URL()
			if err != nil {
				return errors.Newf("invalid IRI filter element %s when loading collections", iri)
			}
			if string(root) != url.Host {
				return errors.Newf("trying to load from non-local root bucket %s", url.Host)
			}
			path = url.Path
		}
		// Assume bucket exists and has keys
		b, path, err := descendInBucket(rb, path, false)
		if err != nil {
			return err
		}
		if b == nil {
			return errors.Errorf("Invalid bucket %s/%s", root, path)
		}

		c := b.Cursor()
		if c == nil {
			return errors.Errorf("Invalid bucket cursor %s/%s", root, path)
		}
		isObjectKey := func(k []byte) bool {
			return string(k) == objectKey || string(k) == metaDataKey
		}
		if path != "" {
			// when we get a non empty path from descendIntoBucket we try to load it as a valid object key
			prefix := []byte(path)
			for key, raw := c.Seek(prefix); key != nil && bytes.HasPrefix(key, prefix); key, raw = c.Next() {
				it, err := filterIt(key, raw, f)
				if err != nil {
					// log error and continue
					continue
				}
				if it != nil {
					col = append(col, it)
				}
			}
		} else {
			// if no path was returned from descendIntoBucket we iterate over all keys in the current bucket
			for key, raw := c.First(); key != nil; key, raw = c.Next() {
				if !isObjectKey(key) {
					b := b.Bucket(key) // FIXME(marius): I guess this should not happen (as descendIntoBucket should 'descend' into 'path' if it's a valid bucket)
					if b == nil {
						return nil
					}
					raw = b.Get([]byte(objectKey))
				}
				it, err := filterIt(key, raw, f)
				if err != nil {
					// log error and continue
					continue
				}
				if it != nil {
					if itCol, ok := it.(as.ItemCollection); ok {
						col = itCol
					} else {
						col = append(col, it)
					}
				}
			}
		}
		return nil
	})

	return col, uint(len(col)), err
}

func filterActivity(it as.Item, f s.Filterable) (bool, as.Item) {
	ff, ok := f.(s.FilterableActivity)
	if !ok {
		return true, it
	}
	ok, _ = filterObject(it, f)
	act, _ := as.ToActivity(it)
	actors := ff.Actors()
	if len(actors) > 0 {
		exists := false
		for _, actor := range actors {
			if actor.GetLink() == act.Object {
				exists = true
				break
			}
			if exists {
				break
			}
		}
		if !exists {
			return false, nil
		}
	}
	objects := ff.Objects()
	if len(objects) > 0 {
		exists := false
		for _, object := range objects {
			if object.GetLink() == act.Object {
				exists = true
				break
			}
			if exists {
				break
			}
		}
		if !exists {
			return false, nil
		}
	}
	targets := ff.Targets()
	if len(targets) > 0 {
		exists := false
		for _, target := range targets {
			if target.GetLink() == act.Target {
				exists = true
				break
			}
			if exists {
				break
			}
		}
		if !exists {
			return false, nil
		}
	}
	return ok, act
}

func filterPerson(it as.Item, f s.Filterable) (bool, as.Item) {
	ff, ok := f.(s.FilterableObject)
	if !ok {
		return true, it
	}
	ob, _ := auth.ToPerson(it)
	names := ff.Names()
	if len(names) > 0 {
		exists := false
		for _, name := range names {
			if len(ob.PreferredUsername) > 0 {
				for _, nn := range ob.PreferredUsername {
					if strings.ToLower(nn.Value) == strings.ToLower(name) {
						exists = true
						break
					}
					if exists {
						break
					}
				}
				for _, nn := range ob.Name {
					if strings.ToLower(nn.Value) == strings.ToLower(name) {
						exists = true
						break
					}
					if exists {
						break
					}
				}
			}
		}
		if !exists {
			return false, nil
		}
	}
	authors := ff.AttributedTo()
	if len(authors) > 0 && !authors.Contains(ob.AttributedTo.GetLink()) {
		return false, nil
	}
	inReply := ff.InReplyTo()
	if len(inReply) > 0 && !inReply.Contains(ob.InReplyTo.GetLink()) {
		return false, nil
	}
	audFilter := ff.Audience()
	if len(audFilter) > 0 {
		exists := false
		for _, aud := range ob.Audience {
			if audFilter.Contains(aud.GetLink()) {
				exists = true
				break
			}
		}
		for _, to := range ob.To {
			if audFilter.Contains(to.GetLink()) {
				exists = true
				break
			}
		}
		for _, bto := range ob.Bto {
			if audFilter.Contains(bto.GetLink()) {
				exists = true
				break
			}
		}
		if !exists {
			return false, nil
		}
	}
	medTypes := ff.MediaTypes()
	if len(medTypes) > 0 {
		exists := false
		for _, typ := range medTypes {
			if typ == ob.MediaType {
				exists = true
			}
		}
		if !exists {
			return false, nil
		}
	}
	return ok, ob
}

func filterObject(it as.Item, f s.Filterable) (bool, as.Item) {
	ff, ok := f.(s.FilterableObject)
	if !ok {
		return true, it
	}
	ob, _ := auth.ToObject(it)
	names := ff.Names()
	if len(names) > 0 {
		exists := false
		for _, name := range names {
			for _, nn := range ob.Name {
				if strings.ToLower(nn.Value) == strings.ToLower(name) {
					exists = true
					break
				}
				if exists {
					break
				}
			}
		}
		if !exists {
			return false, nil
		}
	}
	authors := ff.AttributedTo()
	if len(authors) > 0 && !authors.Contains(ob.AttributedTo.GetLink()) {
		return false, nil
	}
	inReply := ff.InReplyTo()
	if len(inReply) > 0 && !inReply.Contains(ob.InReplyTo.GetLink()) {
		return false, nil
	}
	audFilter := ff.Audience()
	if len(audFilter) > 0 {
		exists := false
		for _, aud := range ob.Audience {
			if audFilter.Contains(aud.GetLink()) {
				exists = true
				break
			}
		}
		for _, to := range ob.To {
			if audFilter.Contains(to.GetLink()) {
				exists = true
				break
			}
		}
		for _, bto := range ob.Bto {
			if audFilter.Contains(bto.GetLink()) {
				exists = true
				break
			}
		}
		if !exists {
			return false, nil
		}
	}
	medTypes := ff.MediaTypes()
	if len(medTypes) > 0 {
		exists := false
		for _, typ := range medTypes {
			if typ == ob.MediaType {
				exists = true
			}
		}
		if !exists {
			return false, nil
		}
	}
	return true, ob
}

func filterIt(key, raw []byte, f s.Filterable) (as.Item, error) {
	// key can be one of 'objectKey', 'metaKey', or collection names: 'inbox', 'outbox', 'liked', samd.
	if string(key) == metaDataKey {
		// this is an error
		return nil, errors.Errorf("trying to load invalid data %s", key)
	}
	if handlers.ValidCollection(string(key)) {
		// if the current key represents a valid collection name,
		// should have been handled by the descendIntoBucket, so this is an error
		return nil, errors.Errorf("trying to load invalid data %s", key)
	}
	if raw == nil || len(raw) == 0 {
		// TODO(marius): log this instead of stopping the iteration and returning an error
		return nil, errors.Errorf("empty raw item")
	}

	var it as.Item
	var err error
	if handlers.ValidCollection(path.Base(f.GetLink().String())) {
		col := make([]as.IRI,0)
		err = jsonld.Unmarshal(raw, &col)
		iris := make(as.ItemCollection, len(col))
		for i, iri := range col {
			iris[i] = &iri
		}
		it = iris
	} else {
		it, err = as.UnmarshalJSON(raw)
	}
	if err != nil {
		// TODO(marius): log this instead of stopping the iteration and returning an error
		return nil, errors.Annotatef(err, "unable to unmarshal raw item")
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
	}
	var valid bool
	if as.ActivityTypes.Contains(it.GetType()) {
		valid, _ = filterActivity(it, f)
	} else if as.ActorTypes.Contains(it.GetType()) {
		valid, _ = filterPerson(it, f)
	} else {
		valid, _ = filterObject(it, f)
	}
	if !valid {
		return nil, nil
	}
	return it, nil
}

// Load
func (r *repo) Load(f s.Filterable) (as.ItemCollection, uint, error) {
	var err error
	err = r.Open()
	if err != nil {
		return nil, 0, err
	}
	defer r.Close()
	return loadFromBucket(r.d, r.root, f)
}

// LoadActivities
func (r *repo) LoadActivities(f s.Filterable) (as.ItemCollection, uint, error) {
	return r.Load(f)
}

// LoadObjects
func (r *repo) LoadObjects(f s.Filterable) (as.ItemCollection, uint, error) {
	return r.Load(f)
}

// LoadActors
func (r *repo) LoadActors(f s.Filterable) (as.ItemCollection, uint, error) {
	return r.Load(f)
}

func descendInBucket(root *bolt.Bucket, path string, create bool) (*bolt.Bucket, string, error) {
	if root == nil {
		return nil, path, errors.Newf("Trying to descend into nil bucket")
	}
	if len(path) == 0 {
		return nil, path, errors.Newf("Trying to descend into nil bucket tree")
	}
	buckets := strings.Split(path, "/")

	lvl := 0
	b := root
	// descend the bucket tree up to the last found bucket
	for _, name := range buckets {
		lvl++
		if len(name) == 0 {
			continue
		}
		if b == nil {
			return root, path, errors.Errorf("trying to load from nil bucket")
		}
		var cb *bolt.Bucket
		if create {
			cb, _ = b.CreateBucketIfNotExists([]byte(name))
		} else {
			cb = b.Bucket([]byte(name))
		}
		if cb == nil {
			lvl--
			break
		}
		b = cb
	}
	path = strings.Join(buckets[lvl:], "/")

	return b, path, nil
}

// LoadCollection
func (r *repo) LoadCollection(f s.Filterable) (as.CollectionInterface, error) {
	var err error
	err = r.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var ret as.CollectionInterface
	iri := f.GetLink()
	url, err := iri.URL()
	if err != nil {
		r.errFn(nil, "invalid IRI filter element %s when loading collections", iri)
	}
	if string(r.root) != url.Host {
		r.errFn(nil, "trying to load from non-local root bucket %s", url.Host)
	}
	col := &as.OrderedCollection{}
	col.ID = as.ObjectID(iri)
	col.Type = as.OrderedCollectionType

	elements, count, err := loadFromBucket(r.d, r.root, f)
	if err != nil {
		return col, errors.Annotatef(err, "Unable to load elements")
	}
	if count == 0 {
		return col, nil
	}
	for _, it := range elements {
		if err = col.Append(it); err == nil {
			col.TotalItems++
		}
	}

	ret = col
	return ret, err
}

const objectKey = "__raw"
const metaDataKey = "__meta_data"

func delete(r *repo, it as.Item) (as.Item, error) {
	f := ap.Filters{
		IRI: it.GetLink(),
	}
	if it.IsObject() {
		f.Type = []as.ActivityVocabularyType{it.GetType()}
	}
	var cnt uint
	var found as.ItemCollection
	found, cnt, _ = loadFromBucket(r.d, r.root, &f)
	if cnt == 0 {
		err := errors.NotFoundf("%s in either actors or objects", it.GetLink())
		return it, err
	}
	old := found.First()

	// TODO(marius): add some mechanism for marking the collections as read-only
	t := as.Tombstone{
		Parent: as.Parent{
			ID:   as.ObjectID(it.GetLink()),
			Type: as.TombstoneType,
			To: as.ItemCollection{
				auth.ActivityStreamsPublicNS,
			},
		},
		Deleted:    time.Now().UTC(),
		FormerType: old.GetType(),
	}
	return save(r, t)
}

func (r *repo) CreateCollection(col as.CollectionInterface) (as.CollectionInterface, error) {
	var err error
	err = r.Open()
	if err != nil {
		return col, err
	}
	defer r.Close()

	url, err := col.GetLink().URL()
	if err != nil {
		return col, err
	}
	cPath := url.Path
	c := path.Base(cPath)
	err = r.d.Update(func(tx *bolt.Tx) error {
		root := tx.Bucket(r.root)
		b, _, err := descendInBucket(root, cPath, true)
		if err != nil {
			return err
		}
		return b.Put([]byte(c), nil)
	})
	return col, err
}

func save(r *repo, it as.Item) (as.Item, error) {
	url, err := it.GetLink().URL()
	if err != nil {
		return it, errors.Annotatef(err, "invalid IRI")
	}
	path := url.Path

	var uuid string
	err = r.d.Update(func(tx *bolt.Tx) error {
		root := tx.Bucket(r.root)
		if root == nil {
			return errors.Errorf("Invalid bucket %s", r.root)
		}
		if !root.Writable() {
			return errors.Errorf("Non writeable bucket %s", r.root)
		}
		var b *bolt.Bucket
		b, uuid, err = descendInBucket(root, path, true)
		if err != nil {
			return errors.Newf("Unable to find %s in root bucket", path)
		}
		if !b.Writable() {
			return errors.Errorf("Non writeable bucket %s", path)
		}
		if uuid != "" {
			b, err = b.CreateBucket([]byte(uuid))
			if err != nil {
				return errors.Errorf("could not create item bucket entry: %s", err)
			}
		}
		return nil
	})

	err = r.d.Update(func(tx *bolt.Tx) error {
		root := tx.Bucket(r.root)
		if root == nil {
			return errors.Errorf("Invalid bucket %s", r.root)
		}
		if !root.Writable() {
			return errors.Errorf("Non writeable bucket %s", r.root)
		}
		var b *bolt.Bucket
		b, uuid, err = descendInBucket(root, path, true)
		if err != nil {
			return errors.Newf("Unable to find %s in root bucket", path)
		}
		if !b.Writable() {
			return errors.Errorf("Non writeable bucket %s", path)
		}
		entryBytes, err := jsonld.Marshal(it)
		if err != nil {
			return errors.Annotatef(err, "could not marshal activity")
		}
		err = b.Put([]byte(objectKey), entryBytes)
		if err != nil {
			return errors.Errorf("could not insert entry: %s", err)
		}

		return nil
	})

	return it, err
}

// SaveActivity
func (r *repo) SaveActivity(it as.Item) (as.Item, error) {
	return r.SaveObject(it)
}

func (r *repo) SaveActor(it as.Item) (as.Item, error) {
	return r.SaveObject(it)
}

// SaveObject
func (r *repo) SaveObject(it as.Item) (as.Item, error) {
	var err error
	err = r.Open()
	if err != nil {
		return it, err
	}
	defer r.Close()

	if it, err = save(r, it); err == nil {
		op := "Updated"
		if it.GetID() == nil {
			op = "Added new"
		}
		r.logFn(nil, "%s %s: %s", op, it.GetType(), it.GetLink())
	}

	// TODO(marius) Move to somewhere else: see the work I started at go-ap/processing/processing.go#225
	if toFw, ok := it.(as.HasRecipients); ok {
		for _, fw := range toFw.Recipients() {
			colIRI := fw.GetLink()
			if r.IsLocalIRI(colIRI) {
				// TODO(marius): this needs some logic to add to the correct Inbox/Outbox collection
				//   depending on the fact if it's a C2S or an S2S request.
				//   For now we hard-code it to C2S => Outbox.
				if !handlers.ValidCollection(path.Base(colIRI.String())) {
					colIRI = as.IRI(fmt.Sprintf("%s/%s", colIRI, handlers.Outbox))
				}
				// we shadow the err variable intentionally so it does not propagate upper to the call stack
				if errFw := r.AddToCollection(colIRI, it); err != nil {
					r.errFn(logrus.Fields{"IRI": it.GetLink(), "collection": colIRI, "error": errFw}, "unable to add to collection")
				}
			}
		}
	}
	return it, err
}

// IsLocalIRI shows if the received IRI belongs to the current instance
func (r repo) IsLocalIRI(i as.IRI) bool {
	if _, err := url.Parse(i.String()); err != nil {
		// not an url
		r.errFn(logrus.Fields{
			"IRI": i,
		}, "Invalid url")
		return false
	}
	return strings.Contains(i.String(), r.baseURL)
}

func (r *repo) AddToCollection(col as.IRI, it as.Item) error {
	if it == nil {
		return errors.Newf("unable to add nil element to collection")
	}
	if len(col) == 0 {
		return errors.Newf("unable to find collection")
	}
	if len(it.GetLink()) == 0 {
		return errors.Newf("Invalid create collection does not have a valid IRI")
	}
	if !r.IsLocalIRI(col.GetLink()) {
		return errors.Newf("Unable to save to non local collection %s", col)
	}
	url, err := col.URL()
	if err != nil {
		return errors.Annotatef(err, "invalid IRI")
	}
	path := url.Path

	return r.d.Update(func(tx *bolt.Tx) error {
		var rem string
		root := tx.Bucket(r.root)
		if root == nil {
			return errors.Errorf("Invalid bucket %s", r.root)
		}
		if !root.Writable() {
			return errors.Errorf("Non writeable bucket %s", r.root)
		}
		var b *bolt.Bucket
		b, rem, err = descendInBucket(root, path, true)
		if err != nil {
			return errors.Newf("Unable to find %s in root bucket", path)
		}
		if rem == "" {
			rem = objectKey
		}
		if !b.Writable() {
			return errors.Errorf("Non writeable bucket %s", path)
		}
		var iris []as.IRI
		raw := b.Get([]byte(rem))
		if len(raw) > 0 {
			err := jsonld.Unmarshal(raw, iris)
			if err != nil {
				return errors.Newf("Unable to unmarshal entries in collection %s", path)
			}
		}
		iris = append(iris, it.GetLink())
		raw, err := jsonld.Marshal(iris)
		if err != nil {
			return errors.Newf("Unable to marshal entries in collection %s", path)
		}
		err = b.Put([]byte(rem), raw)
		if err != nil {
			return errors.Newf("Unable to save entries to collection %s", path)
		}

		return err
	})
}

func (r *repo) UpdateActor(it as.Item) (as.Item, error) {
	return r.UpdateObject(it)
}

// UpdateObject
func (r *repo) UpdateObject(it as.Item) (as.Item, error) {
	return r.SaveObject(it)
}

func (r *repo) DeleteActor(it as.Item) (as.Item, error) {
	return r.DeleteObject(it)
}

// DeleteObject
func (r *repo) DeleteObject(it as.Item) (as.Item, error) {
	var err error
	err = r.Open()
	if err != nil {
		return it, err
	}
	defer r.Close()
	var bucket string
	if as.ActivityTypes.Contains(it.GetType()) {
		bucket = bucketActivities
	} else if as.ActorTypes.Contains(it.GetType()) {
		bucket = bucketActors
	} else {
		bucket = bucketObjects
	}
	if it, err = delete(r, it); err == nil {
		r.logFn(nil, "Added new %s: %s", bucket[:len(bucket)-1], it.GetLink())
	}
	return it, err
}

// GenerateID
func (r *repo) GenerateID(it as.Item, by as.Item) (as.ObjectID, error) {
	typ := it.GetType()
	var partOf string
	if as.ActivityTypes.Contains(typ) {
		partOf = fmt.Sprintf("%s/activities", r.baseURL)
	} else if as.ActorTypes.Contains(typ) || typ == as.ActorType {
		partOf = fmt.Sprintf("%s/actors", r.baseURL)
	} else if as.ObjectTypes.Contains(typ) {
		partOf = fmt.Sprintf("%s/objects", r.baseURL)
	}
	return ap.GenerateID(it, partOf, by)
}

func (r *repo) Open() error {
	var err error
	r.d, err = bolt.Open(r.path, 0600, nil)
	if err != nil {
		return errors.Annotatef(err, "could not open db %s", r.path)
	}
	return nil
}

// Close closes the boltdb database if possible.
func (r *repo) Close() error {
	if r.d == nil {
		return nil
	}
	return r.d.Close()
}

// PasswordSet
func (r *repo) PasswordSet(it as.Item, pw []byte) error {
	url, err := it.GetLink().URL()
	if err != nil {
		return errors.Annotatef(err, "invalid IRI")
	}
	err = r.Open()
	if err != nil {
		return err
	}
	defer r.Close()

	path := url.Path

	type meta struct {
		Pw []byte `json:"pw"`
	}

	err = r.d.Update(func(tx *bolt.Tx) error {
		root := tx.Bucket(r.root)
		if root == nil {
			return errors.Errorf("Invalid bucket %s", r.root)
		}
		if !root.Writable() {
			return errors.Errorf("Non writeable bucket %s", r.root)
		}
		var b *bolt.Bucket
		b, _, err = descendInBucket(root, path, true)
		if err != nil {
			return errors.Newf("Unable to find %s in root bucket", path)
		}
		if !b.Writable() {
			return errors.Errorf("Non writeable bucket %s", path)
		}

		m := meta{
			Pw: pw,
		}
		entryBytes, err := jsonld.Marshal(m)
		if err != nil {
			return errors.Annotatef(err, "could not marshal metadata")
		}
		err = b.Put([]byte(metaDataKey), entryBytes)
		if err != nil {
			return errors.Errorf("could not insert entry: %s", err)
		}
		return nil
	})

	return err
}
