package storage

import (
	"fmt"
	"github.com/go-ap/activitypub/client"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/handlers"
	"github.com/go-ap/jsonld"
	s "github.com/go-ap/storage"
	uuid2 "github.com/google/uuid"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/pgtype"
	"github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
	"strings"
	"time"
)

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

type ActorLoader interface {
	LoadActors(f s.Filterable) (as.ItemCollection, int, error)
}

type loader struct {
	baseURL string
	conn    *pgx.ConnPool
	d       client.Client
	logFn   loggerFn
	errFn   loggerFn
}

type loggerFn func(logrus.Fields, string, ...interface{})

// IsLocalIRI shows if the received IRI belongs to the current instance
func (l loader) IsLocalIRI(i as.IRI) bool {
	return strings.Contains(i.String(), l.baseURL)
}

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

func (l loader) LoadActivities(ff s.Filterable) (as.ItemCollection, int, error) {
	f, ok := ff.(*activitypub.Filters)
	if !ok {
		return nil, 0, errors.Newf("invalid activitypub filters")
	}
	return loadFromDb(l.conn, "activities", f)
}

func (l loader) LoadActors(ff s.Filterable) (as.ItemCollection, int, error) {
	f, ok := ff.(*activitypub.Filters)
	if !ok {
		return nil, 0, errors.Newf("invalid activitypub filters")
	}
	return loadFromDb(l.conn, "actors", f)
}

func (l loader) LoadObjects(ff s.Filterable) (as.ItemCollection, int, error) {
	f, ok := ff.(*activitypub.Filters)
	if !ok {
		return nil, 0, errors.Newf("invalid activitypub filters")
	}
	return loadFromDb(l.conn, "objects", f)
}

func (l loader) LoadCollection(ff s.Filterable) (as.CollectionInterface, error) {
	f, ok := ff.(*activitypub.Filters)
	if !ok {
		return nil, errors.Newf("invalid activitypub filters")
	}
	clauses, values := f.GetWhereClauses()

	var ret as.CollectionInterface

	sel := fmt.Sprintf("SELECT id, iri, created_at::timestamptz, type, count, elements FROM collections WHERE %s ORDER BY created_at DESC LIMIT 1", strings.Join(clauses, " AND "))
	rows, err := l.conn.Query(sel, values...)
	defer rows.Close()
	if err != nil {
		if err == pgx.ErrNoRows {
			return ret, nil
		}
		return ret, errors.Annotatef(err, "unable to run select")
	}
	if err := rows.Err(); err != nil {
		return ret, errors.Annotatef(err, "unable to run select")
	}

	var count int
	// Iterate through the result set
	for rows.Next() {
		var id int64
		var iri string
		var created pgtype.Timestamptz
		var typ string
		var elements []string
		err = rows.Scan(&id, &iri, &created, &typ, &count, &elements)
		if err != nil {
			return ret, errors.Annotatef(err, "scan values error")
		}
		l.logFn(logrus.Fields{
			"id": id,
			"iri": iri,
			"created_at": created,
			"type": typ,
			"count": count,
			"elements": elements,
		}, "loaded fields")

		if as.ActivityVocabularyType(typ) == as.CollectionType {
			col := &as.Collection{}
			col.ID = as.ObjectID(iri)
			col.Type = as.CollectionType
			ret = col
		}
		if as.ActivityVocabularyType(typ) == as.OrderedCollectionType {
			col := &as.OrderedCollection{}
			col.ID = as.ObjectID(iri)
			col.Type = as.OrderedCollectionType
			ret = col
		}
		f.ItemKey = f.ItemKey[:0]
		for _, elem := range elements {
			f.ItemKey = append(f.ItemKey, activitypub.Hash(elem))
		}
	}
	if ret == nil {
		return ret, errors.Newf("could not load '%s' collection", f.Collection)
	}

	table := "objects"
	var items as.ItemCollection
	if f.Collection == "actors" {
		table = "actors"
	}
	if f.Collection == "activities" {
		table = "activities"
	}
	f.ItemKey = f.ItemKey[:0]
	var total int
	items, total, err = loadFromDb(l.conn, table, f)
	if err == nil && total > 0 {
		for _, it := range items {
			ret.Append(it)
		}
	}
	ret, err = activitypub.GetPaginatedCollection(ret, uint(count), f)
	return ret, err
}

func (l loader) Load(ff s.Filterable) (as.ItemCollection, int, error) {
	return nil, 0, errors.NotImplementedf("not implemented loader.Load()")
}

func loadFromDb(conn *pgx.ConnPool, table string, f activitypub.Filterable) (as.ItemCollection, int, error) {
	clauses, values := f.GetWhereClauses()
	total := 0

	sel := fmt.Sprintf("SELECT id, key, iri, created_at::timestamptz, type, raw FROM %s WHERE %s ORDER BY raw->>'published' DESC %s", table, strings.Join(clauses, " AND "), f.GetLimit())
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
		var key activitypub.Hash
		var iri string
		var created pgtype.Timestamptz
		var typ string
		var raw []byte
		err = rows.Scan(&id, &key, &iri, &created, &typ, &raw)
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

	it, err = processActivity(l, it)
	it = activitypub.FlattenProperties(it)

	it, err = l.SaveObject(it)
	if err != nil {
		l.errFn(logrus.Fields{"IRI": it.GetLink()}, "unable to save activity")
		return it, err
	}

	return it, err
}

func getCollectionID(actor as.Item, c handlers.CollectionType) as.ObjectID {
	return as.ObjectID(fmt.Sprintf("%s/%s", actor.GetLink(), c))
}

func getCollectionIRI(actor as.Item, c handlers.CollectionType) as.IRI {
	return as.IRI(fmt.Sprintf("%s/%s", actor.GetLink(), c))
}

func processActivity(l loader, it as.Item) (as.Item, error) {
	var err error

	// First we process the activity to effect whatever changes we need to on the activity properties.
	act, err := activitypub.ToActivity(it)
	if as.ContentManagementActivityTypes.Contains(it.GetType()) {
		act, err = l.ContentManagementActivity(act)
		if err == nil {
			return act, nil
		}
	}
	if as.CollectionManagementActivityTypes.Contains(it.GetType()) {
		act, err = l.CollectionManagementActivity(act)
		if err == nil {
			return act, nil
		}
	}
	if as.ReactionsActivityTypes.Contains(it.GetType()) {
		act, err = l.ReactionsActivity(act)
		if err == nil {
			return act, nil
		}
	}
	if as.EventRSVPActivityTypes.Contains(it.GetType()) {
		act, err = l.EventRSVPActivity(act)
		if err == nil {
			return act, nil
		}
	}
	if as.GroupManagementActivityTypes.Contains(it.GetType()) {
		act, err = l.GroupManagementActivity(act)
		if err == nil {
			return act, nil
		}
	}
	if as.ContentExperienceActivityTypes.Contains(it.GetType()) {
		act, err = l.ContentExperienceActivity(act)
		if err == nil {
			return act, nil
		}
	}
	if as.GeoSocialEventsActivityTypes.Contains(it.GetType()) {
		act, err = l.GeoSocialEventsActivity(act)
		if err == nil {
			return act, nil
		}
	}
	if as.NotificationActivityTypes.Contains(it.GetType()) {
		act, err = l.NotificationActivity(act)
		if err == nil {
			return act, nil
		}
	}
	if as.QuestionActivityTypes.Contains(it.GetType()) {
		act, err = l.QuestionActivity(act)
		if err == nil {
			return act, nil
		}
	}
	if as.RelationshipManagementActivityTypes.Contains(it.GetType()) && act.Object.GetType() == as.RelationshipType {
		act, err = l.RelationshipManagementActivity(act)
		if err == nil {
			return act, errors.Annotatef(err, "%s activity processing failed", act.Type)
		}
	}
	if as.NegatingActivityTypes.Contains(it.GetType()) {
		act, err = l.NegatingActivity(act)
		if err == nil {
			return act, nil
		}
	}
	if as.OffersActivityTypes.Contains(it.GetType()) {
		act, err = l.OffersActivity(act)
		if err == nil {
			return act, nil
		}
	}
	return it, err
}

func ( l loader) createActorCollection(actor as.Item, c handlers.CollectionType) (as.CollectionInterface, error) {
	col := as.OrderedCollection{
		Parent: as.Parent {
			ID: getCollectionID(actor, c),
			Type: as.OrderedCollectionType,
		},
	}
	return createCollection(l, &col)
}
func ( l loader) createObjectCollection(object as.Item, c handlers.CollectionType) (as.CollectionInterface, error) {
	col := as.OrderedCollection{
		Parent: as.Parent {
			ID: getCollectionID(object, c),
			Type: as.OrderedCollectionType,
		},
	}
	return createCollection(l, &col)
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
		if _, cnt, _ := loadFromDb(l.conn, table, &activitypub.Filters{
			ItemKey: []activitypub.Hash{activitypub.Hash(it.GetLink().String())},
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
	it, err = saveToDb(l, table, it)
	if err != nil {
		return it, err
	}

	err = addToCollection(l, getCollectionIRI(as.IRI(l.baseURL), handlers.CollectionType(table)), it)
	if err != nil {
		l.errFn(logrus.Fields{"IRI": it.GetLink(), "collection": table}, "unable to add to collection")
	}

	// TODO(marius) Move to somewhere else
	if toFw, ok := it.(as.HasRecipients); ok {
		for _, fw := range toFw.Recipients() {
			var errs []error
			colIRI := fw.GetLink()
			if l.IsLocalIRI(colIRI) {
				if err := addToCollection(l, colIRI, it); err != nil {
					errs = append(errs, err)
				}
			}
		}
	}

	return it, err
}

func createCollection(l loader, it as.CollectionInterface) (as.CollectionInterface, error) {
	if it == nil {
		return it, errors.Newf("unable to create nil collection")
	}
	if len(it.GetLink()) == 0 {
		return it, errors.Newf("invalid create collection does not have a valid IRI")
	}

	query := fmt.Sprintf("INSERT INTO collections (iri, type, created_at) VALUES ($1, $2, $3::timestamptz);")

	now := time.Now().UTC()
	nowTz := pgtype.Timestamptz{
		Time: now,
		Status: pgtype.Present,
	}
	_, err := l.conn.Exec(query, it.GetLink(), it.GetType(), &nowTz)
	if err != nil {
		l.errFn(logrus.Fields{
			"err": err.Error(),
		}, "query error")
		return it, errors.Annotatef(err, "query error")
	}

	return it, nil
}

func addToCollection(l loader, iri as.IRI, it as.Item) error {
	if it == nil {
		return errors.Newf("unable to add nil element to collection")
	}
	if len(iri) == 0 {
		return errors.Newf("unable to find collection")
	}
	if len(it.GetLink()) == 0 {
		return errors.Newf("invalid create collection does not have a valid IRI")
	}

	query := fmt.Sprintf("UPDATE collections SET updated_at = $1, elements = array_append(elements, $2), count = count+1 WHERE iri = $3;")

	now := time.Now().UTC()
	nowTz := pgtype.Timestamptz{
		Time:   now,
		Status: pgtype.Present,
	}
	t, err := l.conn.Exec(query, &nowTz, it.GetLink(), iri)
	if err != nil {
		l.errFn(logrus.Fields{
			"err": err.Error(),
		}, "query error")
		return errors.Annotatef(err, "query error")
	}
	if t.RowsAffected() != 1 {
		l.errFn(logrus.Fields{
			"rows": t.RowsAffected(),
		}, "query error")
		return errors.Annotatef(err, "query error, invalid updated rows")
	}

	return nil
}

func saveToDb(l loader, table string, it as.Item) (as.Item, error) {
	query := fmt.Sprintf("INSERT INTO %s (key, iri, created_at, type, raw) VALUES ($1, $2, $3::timestamptz, $4, $5::jsonb);", table)

	now := time.Now().UTC()

	uuid := uuid2.New()
	iri := it.GetLink()
	if len(iri) == 0 {
		// TODO(marius): this needs to be in a different place
		iri = as.IRI(fmt.Sprintf("%s/%s/%s", l.baseURL, table, uuid))
		if as.ActivityTypes.Contains(it.GetType()) {
			if a, err := activitypub.ToActivity(it); err == nil {
				a.ID = as.ObjectID(iri)
				a.Published = now
				it = a
			}
		} else if as.ActorTypes.Contains(it.GetType()) {
			if p, err := activitypub.ToPerson(it); err == nil {
				p.ID = as.ObjectID(iri)
				p.Published = now

				if in, err := l.createActorCollection(it, handlers.Inbox); err != nil {
					return it, err
				} else {
					p.Inbox = in.GetLink()
				}
				if out, err := l.createActorCollection(it, handlers.Outbox); err != nil {
					return it, err
				} else {
					p.Outbox = out.GetLink()
				}
				if fers, err := l.createActorCollection(it, handlers.Followers); err != nil {
					return it, err
				} else {
					p.Followers = fers.GetLink()
				}
				if fing, err := l.createActorCollection(it, handlers.Following); err != nil {
					return it, err
				} else {
					p.Following = fing.GetLink()
				}
				if ld, err := l.createActorCollection(it, handlers.Liked); err != nil {
					return it, err
				} else {
					p.Liked = ld.GetLink()
				}
				// TODO(marius): missing likes in go-ap/activitypub actor
				//if ls, err := l.createActorCollection(it, handlers.Likes); err != nil {
				//	return it, err
				//} else {
				//	p.Liked = ls.GetLink()
				//}
				it = p
			}
		} else if as.ObjectTypes.Contains(it.GetType()) {
			if o, err := as.ToObject(it); err == nil {
				o.ID = as.ObjectID(iri)
				o.Published = now
				if repl, err := l.createObjectCollection(it, handlers.Replies); err != nil {
					return it, err
				} else {
					o.Replies = repl.GetLink()
				}
				it = o
			}
		}
	}
	raw, _ := jsonld.Marshal(it)

	nowTz := pgtype.Timestamptz{
		Time: now,
		Status: pgtype.Present,
	}
	_, err := l.conn.Exec(query, uuid, iri, &nowTz, it.GetType(), raw)
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
	iri := it.GetLink()
	if len(iri) == 0 {
		return it, errors.Newf("invalid update item does not have a valid IRI")
	}

	query := fmt.Sprintf("UPDATE %s SET type = $1, updated_at = $2::timestamptz,raw = $3::jsonb WHERE iri = $4;", table)
	now := time.Now().UTC()
	if as.ActorTypes.Contains(it.GetType()) {
		if p, err := activitypub.ToPerson(it); err == nil {
			p.Updated = now
			it = p
		}
	} else if as.ObjectTypes.Contains(it.GetType()) && it.GetType() != as.TombstoneType {
		if o, err := as.ToObject(it); err == nil {
			o.Updated = now
			it = o
		}
	}
	raw, _ := jsonld.Marshal(it)

	nowTz := pgtype.Timestamptz{
		Time: now,
		Status: pgtype.Present,
	}
	_, err := l.conn.Exec(query, it.GetType(), &nowTz, raw, iri)
	if err != nil {
		l.errFn(logrus.Fields{
			"err": err.Error(),
		}, "query error")
		return it, errors.Annotatef(err, "query error")
	}

	return it, nil
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
		err := errors.NotFoundf("Unable to update %s with no ID", label)
		return it, err
	}
	found, cnt, _ := loadFromDb(l.conn, table, &activitypub.Filters{
		ItemKey: []activitypub.Hash{activitypub.Hash(it.GetLink().String())},
		Type:    []as.ActivityVocabularyType{it.GetType()},
	})
	if cnt == 0 {
		err := errors.NotFoundf("%s %s", it.GetLink(), label)
		l.errFn(logrus.Fields{
			"table": table,
			"type":  it.GetType(),
			"iri":   it.GetLink(),
			"err":   err.Error(),
		}, "unable to find old item")
		return it, err
	}
	old := found[0]
	it, err = activitypub.UpdateItemProperties(old, it)
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
			"table":  table,
		}, "%s", err.Error())
	}

	return it, err
}

func (l loader) DeleteObject(it as.Item) (as.Item, error) {
	if it == nil {
		return it, errors.Newf("not saving nil item")
	}
	var table string

	if as.ActivityTypes.Contains(it.GetType()) {
		return nil, errors.Newf("unable to delete activity")
	} else if as.ActorTypes.Contains(it.GetType()) {
		table = "actors"
	} else {
		table = "objects"
	}

	f := activitypub.Filters{
		ItemKey: []activitypub.Hash{activitypub.Hash(it.GetLink().String())},
	}
	if it.IsObject() {
		f.Type = []as.ActivityVocabularyType{it.GetType()}
	}
	var cnt int
	var found as.ItemCollection
	found, cnt, _ = loadFromDb(l.conn, table, &f)
	if cnt == 0 {
		if table == "objects" {
			table = "actors"
		}
		// try other table
		found, cnt, _ = loadFromDb(l.conn, table, &f)
	}
	if cnt == 0 {
		err := errors.NotFoundf("%s in either actors or objects", it.GetLink())
		l.errFn(logrus.Fields{
			"table": table,
			"type":  it.GetType(),
			"iri":   it.GetLink(),
			"err":   err.Error(),
		}, "unable to find old item")
		return it, err
	}
	old := found.First()

	t := as.Tombstone{
		Parent: as.Parent{
			ID:   as.ObjectID(it.GetLink()),
			Type: as.TombstoneType,
			To: as.ItemCollection{
				as.IRI(activitypub.Public),
			},
		},
		Deleted:    time.Now().UTC(),
		FormerType: old.GetType(),
	}

	return l.updateItem(table, t)
}

// ContentManagementActivity processes matching activities
func (l loader) ContentManagementActivity(act *activitypub.Activity) (*activitypub.Activity, error) {
	var err error
	if act.Object != nil {
		switch act.Type {
		case as.CreateType:
			// TODO(marius) Add function as.AttributedTo(it as.Item, auth as.Item)
			if o, err := activitypub.ToObject(act.Object); err == nil {
				// See https://www.w3.org/TR/activitypub/#create-activity-outbox
				// Copying the actor's IRI to the object's AttributedTo
				o.AttributedTo = act.Actor.GetLink()

				aRec := act.Recipients()
				// Copying the activity's recipients to the object's
				o.Audience = aRec
				// Copying the object's recipients to the activity's audience
				act.Audience = o.Recipients()

				act.Object = o
			}
			act.Object, err = l.SaveObject(act.Object)
		case as.UpdateType:
			// TODO(marius): Move this piece of logic to the validation mechanism
			if len(act.Object.GetLink()) == 0 {
				return act, errors.Newf("unable to update object without a valid object id")
			}
			act.Object, err = l.UpdateObject(act.Object)
		case as.DeleteType:
			// TODO(marius): Move this piece of logic to the validation mechanism
			if len(act.Object.GetLink()) == 0 {
				return act, errors.Newf("unable to update object without a valid object id")
			}
			act.Object, err = l.DeleteObject(act.Object)
		}
		if err != nil && !IsDuplicateKey(err) {
			l.errFn(logrus.Fields{"IRI": act.GetLink(), "type": act.Type}, "unable to save activity's object")
			return act, err
		}
	}
	return act, err
}

// ReactionsActivity processes matching activities
func (l loader) ReactionsActivity(act *activitypub.Activity) (*activitypub.Activity, error) {
	var err error
	if act.Object != nil {
		switch act.Type {
		case as.BlockType:
		case as.AcceptType:
			// TODO(marius): either the actor or the object needs to be local for this action to be valid
			// in the case of C2S... the actor needs to be local
			// in the case of S2S... the object is
		case as.DislikeType:
		case as.FlagType:
		case as.IgnoreType:
		case as.LikeType:
		case as.RejectType:
		case as.TentativeAcceptType:
		case as.TentativeRejectType:
		}
	}
	return act, err
}

// CollectionManagementActivity processes matching activities
func (l loader) CollectionManagementActivity(act *activitypub.Activity) (*activitypub.Activity, error) {
	// TODO(marius):
	return nil, errors.Errorf("Not implemented")
}

// EventRSVPActivity processes matching activities
func (l loader) EventRSVPActivity(act *activitypub.Activity) (*activitypub.Activity, error) {
	// TODO(marius):
	return nil, errors.Errorf("Not implemented")
}

// GroupManagementActivity processes matching activities
func (l loader) GroupManagementActivity(act *activitypub.Activity) (*activitypub.Activity, error) {
	// TODO(marius):
	return nil, errors.Errorf("Not implemented")
}

// CollectionManagementActivity processes matching activities
func (l loader) ContentExperienceActivity(act *activitypub.Activity) (*activitypub.Activity, error) {
	// TODO(marius):
	return nil, errors.Errorf("Not implemented")
}

// GeoSocialEventsActivity processes matching activities
func (l loader) GeoSocialEventsActivity(act *activitypub.Activity) (*activitypub.Activity, error) {
	// TODO(marius):
	return nil, errors.Errorf("Not implemented")
}

// NotificationActivity processes matching activities
func (l loader) NotificationActivity(act *activitypub.Activity) (*activitypub.Activity, error) {
	// TODO(marius):
	return nil, errors.Errorf("Not implemented")
}

// QuestionActivity processes matching activities
func (l loader) QuestionActivity(act *activitypub.Activity) (*activitypub.Activity, error) {
	// TODO(marius):
	return nil, errors.Errorf("Not implemented")
}

// RelationshipManagementActivity processes matching activities
func (l loader) RelationshipManagementActivity(act *activitypub.Activity) (*activitypub.Activity, error) {
	// TODO(marius):
	return nil, errors.Errorf("Not implemented")
}

// NegatingActivity processes matching activities
func (l loader) NegatingActivity(act *activitypub.Activity) (*activitypub.Activity, error) {
	// TODO(marius):
	return nil, errors.Errorf("Not implemented")
}

// OffersActivity processes matching activities
func (l loader) OffersActivity(act *activitypub.Activity) (*activitypub.Activity, error) {
	// TODO(marius):
	return nil, errors.Errorf("Not implemented")
}
