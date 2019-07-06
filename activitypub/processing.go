package activitypub

import (
	"fmt"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	"github.com/go-ap/handlers"
	s "github.com/go-ap/storage"
	"time"
)

type errDuplicateKey struct {
	errors.Err
}

func isDuplicateKey(e error) bool {
	_, okp := e.(*errDuplicateKey)
	_, oks := e.(errDuplicateKey)
	return okp || oks
}

func (n errDuplicateKey) Is(e error) bool {
	return isDuplicateKey(e)
}

func wrapErr(err error, s string, args ...interface{}) errors.Err {
	return *errors.Annotatef(err, s, args...)
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

func getCollection(it as.Item, c handlers.CollectionType) as.CollectionInterface {
	return &as.OrderedCollection{
		Parent: as.Parent{
			ID:   as.ObjectID(fmt.Sprintf("%s/%s", it.GetLink(), c)),
			Type: as.OrderedCollectionType,
		},
	}
}

func AddNewObjectCollections(r s.CollectionSaver, it as.Item) (as.Item, error) {
	if as.ActorTypes.Contains(it.GetType()) {
		if p, err := auth.ToPerson(it); err == nil {
			if in, err := r.CreateCollection(getCollection(p, handlers.Inbox)); err != nil {
				return it, errors.Errorf("could not create bucket for collection %s", err)
			} else {
				p.Inbox = in.GetLink()
			}
			if out, err := r.CreateCollection(getCollection(p, handlers.Outbox)); err != nil {
				return it, errors.Errorf("could not create bucket for collection %s", err)
			} else {
				p.Outbox = out.GetLink()
			}
			if fers, err := r.CreateCollection(getCollection(p, handlers.Followers)); err != nil {
				return it, errors.Errorf("could not create bucket for collection %s", err)
			} else {
				p.Followers = fers.GetLink()
			}
			if fing, err := r.CreateCollection(getCollection(p, handlers.Following)); err != nil {
				return it, errors.Errorf("could not create bucket for collection %s", err)
			} else {
				p.Following = fing.GetLink()
			}
			if ld, err := r.CreateCollection(getCollection(p, handlers.Liked)); err != nil {
				return it, errors.Errorf("could not create bucket for collection %s", err)
			} else {
				p.Liked = ld.GetLink()
			}
			if ls, err := r.CreateCollection(getCollection(p, handlers.Likes)); err != nil {
				return it, errors.Errorf("could not create bucket for collection %s", err)
			} else {
				p.Likes = ls.GetLink()
			}
			if sh, err := r.CreateCollection(getCollection(p, handlers.Shares)); err != nil {
				return it, errors.Errorf("could not create bucket for collection %s", err)
			} else {
				p.Shares = sh.GetLink()
			}
			it = p
		}
	} else if as.ObjectTypes.Contains(it.GetType()) {
		if o, err := as.ToObject(it); err == nil {
			if repl, err := r.CreateCollection(getCollection(o, handlers.Replies)); err != nil {
				return it, errors.Errorf("could not create bucket for collection %s", err)
			} else {
				o.Replies = repl.GetLink()
			}
			it = o
		}
	}
	return it, nil
}

// ProcessActivity
func ProcessActivity(r s.Saver, it as.Item) (as.Item, error) {
	var err error

	// TODO(marius): Since we're not failing on the first error, so we can try to process the same type of
	// activity in multiple contexts, we should propagate all the errors to the end, by probably using some
	// errors.Annotatef...

	// First we process the activity to effect whatever changes we need to on the activity properties.
	act, err := as.ToActivity(it)
	if as.ContentManagementActivityTypes.Contains(it.GetType()) && act.Object.GetType() != as.RelationshipType {
		act, err = ContentManagementActivity(r, act)
		if err != nil {
			return it, err
		}
	}
	if as.CollectionManagementActivityTypes.Contains(it.GetType()) {
		act, err = CollectionManagementActivity(r, act)
		if err != nil {
			return it, err
		}
	}
	if as.ReactionsActivityTypes.Contains(it.GetType()) {
		act, err = ReactionsActivity(r, act)
		if err != nil {
			return it, err
		}
	}
	if as.EventRSVPActivityTypes.Contains(it.GetType()) {
		act, err = EventRSVPActivity(r, act)
		if err != nil {
			return it, err
		}
	}
	if as.GroupManagementActivityTypes.Contains(it.GetType()) {
		act, err = GroupManagementActivity(r, act)
		if err != nil {
			return it, err
		}
	}
	if as.ContentExperienceActivityTypes.Contains(it.GetType()) {
		act, err = ContentExperienceActivity(r, act)
		if err != nil {
			return it, err
		}
	}
	if as.GeoSocialEventsActivityTypes.Contains(it.GetType()) {
		act, err = GeoSocialEventsActivity(r, act)
		if err != nil {
			return it, err
		}
	}
	if as.NotificationActivityTypes.Contains(it.GetType()) {
		act, err = NotificationActivity(r, act)
		if err != nil {
			return it, err
		}
	}
	if as.QuestionActivityTypes.Contains(it.GetType()) {
		act, err = QuestionActivity(r, act)
		if err != nil {
			return it, err
		}
	}
	if as.RelationshipManagementActivityTypes.Contains(it.GetType()) && act.Object.GetType() == as.RelationshipType {
		act, err = RelationshipManagementActivity(r, act)
		if err == nil {
			return act, errors.Annotatef(err, "%s activity processing failed", act.Type)
		}
	}
	if as.NegatingActivityTypes.Contains(it.GetType()) {
		act, err = NegatingActivity(r, act)
		if err != nil {
			return it, err
		}
	}
	if as.OffersActivityTypes.Contains(it.GetType()) {
		act, err = OffersActivity(r, act)
		if err != nil {
			return it, err
		}
	}

	iri := it.GetLink()
	if len(iri) == 0 {
		r.GenerateID(it, nil)
	}

	it = FlattenProperties(it)
	return r.SaveActivity(it)
}

// ContentManagementActivity processes matching activities
func ContentManagementActivity(l s.Saver, act *as.Activity) (*as.Activity, error) {
	var err error
	if act.Object == nil {
		return act, errors.NotValidf("Missing object for Activity")
	}
	now := time.Now().UTC()
	switch act.Type {
	case as.CreateType:
		iri := act.Object.GetLink()
		if len(iri) == 0 {
			l.GenerateID(act.Object, act)
		}
		// TODO(marius) Add function as.AttributedTo(it as.Item, auth as.Item)
		if a, err := as.ToActivity(act.Object); err == nil {
			// See https://www.w3.org/TR/ActivityPub/#create-activity-outbox
			// Copying the actor's IRI to the object's AttributedTo
			a.AttributedTo = act.Actor.GetLink()

			// Setting the Generator to the current service if not specified explicitly
			if a.Generator == nil && len(ServiceIRI) > 0 {
				a.Generator = ServiceIRI
			}

			aRec := act.Recipients()
			// Copying the activity's recipients to the object's
			a.Audience = aRec
			// Copying the object's recipients to the activity's audience
			act.Audience = a.Recipients()

			// TODO(marius): Move these to a ProcessObject function
			// Set the published date
			a.Published = now

			act.Object = a
		} else if p, err := auth.ToPerson(act.Object); err == nil {
			// See https://www.w3.org/TR/ActivityPub/#create-activity-outbox
			// Copying the actor's IRI to the object's AttributedTo
			p.AttributedTo = act.Actor.GetLink()

			// Setting the Generator to the current service if not specified explicitly
			if p.Generator == nil && len(ServiceIRI) > 0 {
				p.Generator = ServiceIRI
			}

			aRec := act.Recipients()
			// Copying the activity's recipients to the object's
			p.Audience = aRec
			// Copying the object's recipients to the activity's audience
			act.Audience = p.Recipients()

			// TODO(marius): Move these to a ProcessObject function
			// Set the published date
			p.Published = now

			act.Object = p
		} else if o, err := ToObject(act.Object); err == nil {
			// See https://www.w3.org/TR/ActivityPub/#create-activity-outbox
			// Copying the actor's IRI to the object's AttributedTo
			o.AttributedTo = act.Actor.GetLink()

			// Setting the Generator to the current service if not specified explicitly
			if o.Generator == nil && len(ServiceIRI) > 0 {
				o.Generator = ServiceIRI
			}

			aRec := act.Recipients()
			// Copying the activity's recipients to the object's
			o.Audience = aRec
			// Copying the object's recipients to the activity's audience
			act.Audience = o.Recipients()

			// TODO(marius): Move these to a ProcessObject function
			// Set the published date
			o.Published = now

			act.Object = o
		}

		if colSaver, ok := l.(s.CollectionSaver); ok {
			act.Object, err = AddNewObjectCollections(colSaver, act.Object)
			if err != nil {
				return act, errors.Annotatef(err, "unable to add object collections to object %s", act.Object.GetLink())
			}
		}

		act.Object, err = l.SaveObject(act.Object)
	case as.UpdateType:
		// TODO(marius): Move this piece of logic to the validation mechanism
		if len(act.Object.GetLink()) == 0 {
			return act, errors.Newf("unable to update object without a valid object id")
		}

		ob := act.Object
		var cnt uint
		if as.ActivityTypes.Contains(ob.GetType()) {
			return act, errors.Newf("unable to update activity")
		}

		var found as.ItemCollection
		typ := ob.GetType()
		if loader, ok := l.(s.ActorLoader); ok && as.ActorTypes.Contains(typ) {
			found, cnt, _ = loader.LoadActors(&Filters{
				IRI:  ob.GetLink(),
				Type: []as.ActivityVocabularyType{typ},
			})
		}
		if loader, ok := l.(s.ObjectLoader); ok && as.ObjectTypes.Contains(typ) {
			found, cnt, _ = loader.LoadObjects(&Filters{
				IRI:  ob.GetLink(),
				Type: []as.ActivityVocabularyType{typ},
			})
		}
		if len(ob.GetLink()) == 0 {
			return act, err
		}

		if cnt == 0 {
			return act, errors.NotFoundf("Unable to find %s %s", ob.GetType(), ob.GetLink())
		}
		ob, err = UpdateItemProperties(found.First(), ob)
		if err != nil {
			return act, err
		}

		act.Object, err = l.UpdateObject(ob)
	case as.DeleteType:
		// TODO(marius): Move this piece of logic to the validation mechanism
		if len(act.Object.GetLink()) == 0 {
			return act, errors.Newf("unable to update object without a valid object id")
		}
		act.Object, err = l.DeleteObject(act.Object)
	}
	if err != nil && !isDuplicateKey(err) {
		//l.errFn(logrus.Fields{"IRI": act.GetLink(), "type": act.Type}, "unable to save activity's object")
		return act, err
	}

	// Set the published date
	act.Published = now
	return act, err
}

// ReactionsActivity processes matching activities
func ReactionsActivity(l s.Saver, act *as.Activity) (*as.Activity, error) {
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
func CollectionManagementActivity(l s.Saver, act *as.Activity) (*as.Activity, error) {
	// TODO(marius):
	return nil, errors.Errorf("Not implemented")
}

// EventRSVPActivity processes matching activities
func EventRSVPActivity(l s.Saver, act *as.Activity) (*as.Activity, error) {
	// TODO(marius):
	return nil, errors.Errorf("Not implemented")
}

// GroupManagementActivity processes matching activities
func GroupManagementActivity(l s.Saver, act *as.Activity) (*as.Activity, error) {
	// TODO(marius):
	return nil, errors.Errorf("Not implemented")
}

// CollectionManagementActivity processes matching activities
func ContentExperienceActivity(l s.Saver, act *as.Activity) (*as.Activity, error) {
	// TODO(marius):
	return nil, errors.Errorf("Not implemented")
}

// GeoSocialEventsActivity processes matching activities
func GeoSocialEventsActivity(l s.Saver, act *as.Activity) (*as.Activity, error) {
	// TODO(marius):
	return nil, errors.Errorf("Not implemented")
}

// NotificationActivity processes matching activities
func NotificationActivity(l s.Saver, act *as.Activity) (*as.Activity, error) {
	// TODO(marius):
	return nil, errors.Errorf("Not implemented")
}

// QuestionActivity processes matching activities
func QuestionActivity(l s.Saver, act *as.Activity) (*as.Activity, error) {
	// TODO(marius):
	return nil, errors.Errorf("Not implemented")
}

// RelationshipManagementActivity processes matching activities
func RelationshipManagementActivity(l s.Saver, act *as.Activity) (*as.Activity, error) {
	// TODO(marius):
	return nil, errors.Errorf("Not implemented")
}

// NegatingActivity processes matching activities
func NegatingActivity(l s.Saver, act *as.Activity) (*as.Activity, error) {
	// TODO(marius):
	return nil, errors.Errorf("Not implemented")
}

// OffersActivity processes matching activities
func OffersActivity(l s.Saver, act *as.Activity) (*as.Activity, error) {
	// TODO(marius):
	return nil, errors.Errorf("Not implemented")
}
