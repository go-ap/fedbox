package storage

import (
	"fmt"
	"github.com/go-ap/activitypub/client"
	s "github.com/go-ap/activitypub/storage"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/internal/errors"
	"github.com/go-ap/jsonld"
	uuid2 "github.com/google/uuid"
	"github.com/jackc/pgx"
	"github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
	"strings"
	"time"
)

type Paginator = s.Paginator

type errDuplicateKey struct {
	errors.Err
}

func IsDuplicateKey(e error) bool {
	_, okp := e.(*errDuplicateKey)
	_, oks := e.(errDuplicateKey)
	return okp || oks
}
func (n errDuplicateKey) Is(e error) bool {
	return IsDuplicateKey(e)
}
func wrapErr(err error, s string, args ...interface{}) errors.Err {
	e := errors.Annotatef(err, s, args...)
	asErr := errors.Err{}
	xerrors.As(e, &asErr)
	return asErr
}

var errFn = func(ss string) func(s string, p ...interface{}) errors.Err {
	fn := func(s string, p ...interface{}) errors.Err {
		return wrapErr(nil, fmt.Sprintf("%s: %s", ss, s), p...)
	}
	return fn
}

var ErrDuplicateObject = func(s string, p ...interface{}) errDuplicateKey {
	return errDuplicateKey{wrapErr(nil, fmt.Sprintf("Duplicate key: %s", s), p...)}
}

type Loader interface {
	s.ActivityLoader
	s.ActorLoader
	s.ObjectLoader
	s.ActivitySaver
	s.ActorSaver
	s.ObjectSaver
}

type loader struct {
	baseURL string
	conn    *pgx.ConnPool
	d       client.Client
	logFn   loggerFn
	errFn   loggerFn
}

type loggerFn func(logrus.Fields, string, ...interface{})

func logFn(l logrus.FieldLogger, lvl logrus.Level) loggerFn {
	return func(w logrus.Fields, f string, par ...interface{}) {
		if l != nil {
			l.WithFields(w).Logf(lvl, f, par...)
		}
	}
}

func New(conn *pgx.ConnPool, url string, l logrus.FieldLogger) *loader {
	return &loader{
		conn:    conn,
		baseURL: url,
		d:       client.NewClient(),
		logFn:   logFn(l, logrus.InfoLevel),
		errFn:   logFn(l, logrus.ErrorLevel),
	}
}

func (l loader) Load(f s.Filterable) (as.ItemCollection, int, error) {
	var ff *Filters
	var ok bool
	var total int
	if ff, ok = f.(*Filters); !ok {
		ff = &Filters{}
	}
	ret := make(as.ItemCollection, 0)
	act, total, err := loadFromDb(l.conn, "activities", ff)
	if err != nil {
		ret = append(ret, act...)
	}
	it, total, err := loadFromDb(l.conn, "objects", ff)
	if err != nil {
		ret = append(ret, it...)
	}
	actors, total, err := loadFromDb(l.conn, "actors", ff)
	if err != nil {
		ret = append(ret, actors...)
	}

	return ret, total, err
}
func (l loader) LoadActivities(f s.Filterable) (as.ItemCollection, int, error) {
	var ff *Filters
	var ok bool
	if ff, ok = f.(*Filters); !ok {
		ff = &Filters{}
	}
	return loadFromDb(l.conn, "activities", ff)
}
func (l loader) LoadActors(f s.Filterable) (as.ItemCollection, int, error) {
	var ff *Filters
	var ok bool
	if ff, ok = f.(*Filters); !ok {
		ff = &Filters{}
	}
	return loadFromDb(l.conn, "actors", ff)
}
func (l loader) LoadObjects(f s.Filterable) (as.ItemCollection, int, error) {
	var ff *Filters
	var ok bool
	if ff, ok = f.(*Filters); !ok {
		ff = &Filters{}
	}
	return loadFromDb(l.conn, "objects", ff)
}

func loadFromDb(conn *pgx.ConnPool, table string, f *Filters) (as.ItemCollection, int, error) {
	clauses, values := f.GetWhereClauses()
	total := 0

	sel := fmt.Sprintf("SELECT id, key, iri, created_at, type, raw FROM %s WHERE %s ORDER BY raw->>'published' DESC %s", table, strings.Join(clauses, " AND "), f.GetLimit())
	rows, err := conn.Query(sel, values...)
	defer rows.Close()
	if err != nil {
		if err == pgx.ErrNoRows {
			return as.ItemCollection{}, total, nil
		}
		return nil, total, errors.Annotatef(err, "unable to run select")
	}

	ret := make(as.ItemCollection, 0)
	// Iterate through the result set
	for rows.Next() {
		var id int64
		var key Hash
		var iri string
		var createds string
		var typ string
		var raw []byte
		err = rows.Scan(&id, &key, &iri, &createds, &typ, &raw)
		if err != nil {
			return ret, total, errors.Annotatef(err, "scan values error")
		}

		it, err := as.UnmarshalJSON(raw)
		if err != nil {
			return ret, total, errors.Annotatef(err, "unable to unmarshal raw item")
		}
		ret = append(ret, it)
	}

	selCnt := fmt.Sprintf("SELECT COUNT(id) FROM %s WHERE %s", table, strings.Join(clauses, " AND "))
	if err = conn.QueryRow(selCnt, values...).Scan(&total); err != nil {
		err = errors.Annotatef(err, "unable to count all rows")
	}

	return ret, total, err
}

func (l loader) SaveActivity(it as.Item) (as.Item, error) {
	var err error

	// First we process the activity to effect whatever changes we need to on the activity properties.
	act, err := activitypub.ToActivity(it)
	if err != nil {
		l.errFn(logrus.Fields{"IRI": act.GetLink()}, "unable to load activity")
		return act, err
	}
	if as.ContentManagementActivityTypes.Contains(it.GetType()) {
		act, err = l.ContentManagementActivity(act)
		if err != nil {
			return act, errors.Annotatef(err, "%s activity processing failed", act.Type)
		}
	}
	if as.CollectionManagementActivityTypes.Contains(it.GetType()) {
		// TODO(marius):
	}
	if as.ReactionsActivityTypes.Contains(it.GetType()) {
		// TODO(marius):
	}
	if as.ReactionsActivityTypes.Contains(it.GetType()) {
		// TODO(marius):
	}

	it, err = l.SaveObject(it)
	if err != nil {
		l.errFn(logrus.Fields{"IRI": it.GetLink()}, "unable to save activity")
		return it, err
	}

	return act, err
}

func (l loader) SaveActor(it as.Item) (as.Item, error) {
	return l.SaveObject(it)
}

func (l loader) SaveObject(it as.Item) (as.Item, error) {
	if it == nil {
		return it, errors.Newf("not saving nil item")
	}

	var err error
	var table string
	if as.ActivityTypes.Contains(it.GetType()) {
		table = "activities"
	} else if as.ActorTypes.Contains(it.GetType()) {
		table = "actors"
	} else {
		table = "objects"
	}

	if len(it.GetLink()) > 0 {
		if _, cnt, _ := loadFromDb(l.conn, table, &Filters{
			ItemKey: []Hash{Hash(it.GetLink().String())},
			Type:    []as.ActivityVocabularyType{it.GetType()},
		}); cnt != 0 {
			err := ErrDuplicateObject("%s in table %s", it.GetLink(), table)
			l.errFn(logrus.Fields{
				"table": table,
				"type":  it.GetType(),
				"iri":   it.GetLink(),
				"err":   err.Error(),
			}, "skipping save")
			return it, err
		}
	}
	it, err = l.saveToDb(table, it)
	if err != nil {
		l.errFn(logrus.Fields {
			"action": "insert",
			"table": table,
		}, "%s", err.Error())
	}

	return it, err
}

func (l loader) saveToDb(table string, it as.Item) (as.Item, error) {
	var query string
	query = fmt.Sprintf("INSERT INTO %s (key, iri, type, raw) VALUES ($1, $2, $3, $4);", table)

	uuid := uuid2.New()
	iri := it.GetLink()
	if len(iri) == 0 {
		// TODO(marius): this needs to be in a different place
		iri = as.IRI(fmt.Sprintf("%s/%s/%s", l.baseURL, table, uuid))
		if as.ActivityTypes.Contains(it.GetType()) {
			if a, err := activitypub.ToActivity(it); err == nil {
				a.ID = as.ObjectID(iri)
				a.Published = time.Now()
				it = a
			}
		} else if as.ActorTypes.Contains(it.GetType()) {
			if p, err := activitypub.ToPerson(it); err == nil {
				p.ID = as.ObjectID(iri)
				p.Published = time.Now()
				it = p
			}
		} else if as.ObjectTypes.Contains(it.GetType()) {
			if o, err := as.ToObject(it); err == nil {
				o.ID = as.ObjectID(iri)
				o.Published = time.Now()
				it = o
			}
		}
	}
	raw, _ := jsonld.Marshal(it)

	values := make([]interface{}, 4)
	values[0] = interface{}(uuid)
	values[1] = interface{}(iri)
	values[2] = interface{}(it.GetType())
	values[3] = interface{}(raw)

	_, err := l.conn.Exec(query, values...)
	if err != nil {
		l.errFn(logrus.Fields{
			"err": err.Error(),
		}, "query error")
		return it, errors.Annotatef(err, "query error")
	}

	return it, nil
}

func (l loader) updateItem(table string, it as.Item) (as.Item, error) {
	if table == "activities" {
		return it, errors.Newf("update action invalid, activities are immutable")
	}
	query := fmt.Sprintf("UPDATE %s SET type = $1, raw = $2 WHERE iri = $3;", table)

	iri := it.GetLink()
	if len(iri) == 0 {
		return it, errors.Newf("invalid update item does not have a valid IRI")
	}
	if as.ActorTypes.Contains(it.GetType()) {
		if p, err := activitypub.ToPerson(it); err == nil {
			p.Updated = time.Now()
			it = p
		}
	} else if as.ObjectTypes.Contains(it.GetType()) {
		if o, err := as.ToObject(it); err == nil {
			o.Updated = time.Now()
			it = o
		}
	}
	raw, _ := jsonld.Marshal(it)

	values := make([]interface{}, 3)
	values[0] = interface{}(it.GetType())
	values[1] = interface{}(raw)
	values[2] = interface{}(iri)

	_, err := l.conn.Exec(query, values...)
	if err != nil {
		l.errFn(logrus.Fields{
			"err": err.Error(),
		}, "query error")
		return it, errors.Annotatef(err, "query error")
	}

	return it, nil
}

func UpdatePersonProperties(old, new *activitypub.Person) (*activitypub.Person, error) {
	o, err := UpdateObjectProperties(&old.Parent, &new.Parent)
	old.Parent = *o
	old.Inbox = ReplaceIfItem(old.Inbox, new.Inbox)
	old.Outbox = ReplaceIfItem(old.Outbox, new.Outbox)
	old.Following = ReplaceIfItem(old.Following, new.Following)
	old.Followers = ReplaceIfItem(old.Followers, new.Followers)
	old.Liked = ReplaceIfItem(old.Liked, new.Liked)
	old.PreferredUsername = ReplaceIfNaturalLanguageValues(old.PreferredUsername, new.PreferredUsername)
	return old, err
}

func ReplaceIfItem(old, new as.Item) as.Item {
	if new == nil {
		return old
	}
	return new
}

func ReplaceIfItemCollection(old, new as.ItemCollection) as.ItemCollection {
	if new == nil {
		return old
	}
	return new
}

func ReplaceIfNaturalLanguageValues(old, new as.NaturalLanguageValues) as.NaturalLanguageValues {
	if new == nil {
		return old
	}
	return new
}

func UpdateObjectProperties(old, new *as.Object) (*as.Object, error) {
	old.Name = ReplaceIfNaturalLanguageValues(old.Name, new.Name)
	old.Attachment =  ReplaceIfItem(old.Attachment, new.Attachment)
	old.AttributedTo = ReplaceIfItem(old.AttributedTo, new.AttributedTo)
	old.Audience = ReplaceIfItemCollection(old.Audience, new.Audience)
	old.Content = ReplaceIfNaturalLanguageValues(old.Content, new.Content)
	old.Context =ReplaceIfItem(old.Context, new.Context)
	if len(new.MediaType) > 0 {
		old.MediaType = new.MediaType
	}
	if !new.EndTime.IsZero() {
		old.EndTime = new.EndTime
	}
	old.Generator = ReplaceIfItem(old.Generator, new.Generator)
	old.Icon = ReplaceIfItem(old.Icon, new.Icon)
	old.Image = ReplaceIfItem(old.Image, new.Image)
	old.InReplyTo = ReplaceIfItem(old.InReplyTo, new.InReplyTo)
	old.Location = ReplaceIfItem(old.Location, new.Location)
	old.Preview = ReplaceIfItem(old.Preview, new.Preview)
	if !new.Published.IsZero() {
		old.Published = new.Published
	}
	old.Replies = ReplaceIfItem(old.Replies, new.Replies)
	if !new.StartTime.IsZero() {
		old.StartTime = new.StartTime
	}
	old.Summary = ReplaceIfNaturalLanguageValues(old.Summary, new.Summary)
	old.Tag = ReplaceIfItemCollection(old.Tag, new.Tag)
	if !new.Updated.IsZero() {
		old.Updated = new.Updated
	}
	if new.URL != nil {
		old.URL = new.URL
	}
	old.To = ReplaceIfItemCollection(old.To, new.To)
	old.Bto = ReplaceIfItemCollection(old.Bto, new.Bto)
	old.CC = ReplaceIfItemCollection(old.CC, new.CC)
	old.BCC = ReplaceIfItemCollection(old.BCC, new.BCC)
	if new.Duration == 0 {
		old.Duration = new.Duration
	}
	return old, nil
}

func UpdateItemProperties(old, new as.Item) (as.Item, error) {
	if old == nil {
		return old, errors.Newf("Nil object to update")
	}
	if new == nil {
		return old, errors.Newf("Nil object for update")
	}
	if *old.GetID() != *new.GetID() {
		return old, errors.Newf("Object IDs don't match")
	}
	if old.GetType() != new.GetType() {
		return old, errors.Newf("Invalid object types for update")
	}
	if as.ActorTypes.Contains(old.GetType()) {
		o, err := activitypub.ToPerson(old)
		if err != nil {
			return o, err
		}
		n, err := activitypub.ToPerson(new)
		if err != nil {
			return o, err
		}
		return UpdatePersonProperties(o, n)
	}
	if as.ObjectTypes.Contains(old.GetType()) {
		o, err := as.ToObject(old)
		if err != nil {
			return o, err
		}
		n, err := as.ToObject(new)
		if err != nil {
			return o, err
		}
		return UpdateObjectProperties(o, n)
	}
	return old, errors.Newf("could not process objects with type %s", old.GetType())
}

func (l loader) UpdateObject(it as.Item) (as.Item, error) {
	if it == nil {
		return it, errors.Newf("not saving nil item")
	}
	var err error
	var table string
	label := "item"
	if as.ActivityTypes.Contains(it.GetType()) {
		return nil, errors.Newf("unable to update activity")
	} else if as.ActorTypes.Contains(it.GetType()) {
		label = "actor"
		table = "actors"
	} else {
		label = "object"
		table = "objects"
	}
	if len(it.GetLink()) == 0 {
		err := ErrNotFound(fmt.Sprintf("Unable to update %s with no ID", label))
		return it, err
	}
	found, cnt, _ := loadFromDb(l.conn, table, &Filters{
		ItemKey: []Hash{Hash(it.GetLink().String())},
		Type:    []as.ActivityVocabularyType{it.GetType()},
	})
	if cnt == 0 {
		err := ErrNotFound(fmt.Sprintf("%s %s", it.GetLink(), label))
		l.errFn(logrus.Fields{
			"table": table,
			"type":  it.GetType(),
			"iri":   it.GetLink(),
			"err":   err.Error(),
		}, "unable to find old item")
		return it, err
	}
	old := found[0]
	it, err = UpdateItemProperties(old, it)
	if err != nil {
		l.errFn(logrus.Fields{
			"table": table,
			"type":  old.GetType(),
			"iri":   old.GetLink(),
			"err":   err.Error(),
		}, "invalid")
	}

	it, err = l.updateItem(table, it)
	if err != nil {
		l.errFn(logrus.Fields{
			"action": "update",
			"table": table,
		}, "%s", err.Error())
	}

	return it, err
}

func (l loader) DeleteObject(it as.Item) (as.Item, error) {
	if it == nil {
		return it, errors.Newf("not saving nil item")
	}
	var err error
	var table string
	if as.ActivityTypes.Contains(it.GetType()) {
		return nil, errors.Newf("unable to delete activity")
	} else if as.ActorTypes.Contains(it.GetType()) {
		table = "actors"
	} else {
		table = "objects"
	}

	t := as.Tombstone{
		Parent: as.Parent {
			ID: as.ObjectID(it.GetLink()),
			Type: as.TombstoneType,
			To: as.ItemCollection{
				as.IRI(activitypub.Public),
			},
		},
		Deleted: time.Now().UTC(),
		FormerType: it.GetType(),
	}

	it, err = l.updateItem(table, &t)
	if err != nil {
		l.errFn(logrus.Fields{
			"action": "update",
			"table": table,
		}, "%s", err.Error())
	}

	return it, err
}

func (l loader) ContentManagementActivity(act *activitypub.Activity) (*activitypub.Activity, error) {
	var err error
	if act.Object != nil {
		switch act.Type {
		case as.CreateType:
			act.Object, err = l.SaveObject(act.Object)
		case as.UpdateType:
			// TODO(marius): Move this piece of logic to the validation mechanism
			if len(act.Object.GetLink()) == 0 {
				return act, errors.Newf("unable to update object without a valid object id")
			}
			act.Object, err = l.UpdateObject(act.Object)
		case as.DeleteType:
			act.Object, err = l.DeleteObject(act.Object)
		}
		if err != nil && !IsDuplicateKey(err) {
			l.errFn(logrus.Fields{"IRI": act.GetLink(), "type": act.Type}, "unable to save activity's object")
			return act, err
		}
		act.Object = as.FlattenToIRI(act.Object)
	}
	return act, err
}
