package activitypub

import (
	"fmt"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/errors"
	s "github.com/go-ap/storage"
	"golang.org/x/xerrors"
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

func ProcessActivity(l s.Saver, it as.Item) (as.Item, error) {
	var err error

	// First we process the activity to effect whatever changes we need to on the activity properties.
	act, err := ToActivity(it)
	if as.ContentManagementActivityTypes.Contains(it.GetType()) {
		act, err = ContentManagementActivity(l, act)
		if err == nil {
			return act, nil
		}
	}
	if as.CollectionManagementActivityTypes.Contains(it.GetType()) {
		act, err = CollectionManagementActivity(l, act)
		if err == nil {
			return act, nil
		}
	}
	if as.ReactionsActivityTypes.Contains(it.GetType()) {
		act, err = ReactionsActivity(l, act)
		if err == nil {
			return act, nil
		}
	}
	if as.EventRSVPActivityTypes.Contains(it.GetType()) {
		act, err = EventRSVPActivity(l, act)
		if err == nil {
			return act, nil
		}
	}
	if as.GroupManagementActivityTypes.Contains(it.GetType()) {
		act, err = GroupManagementActivity(l, act)
		if err == nil {
			return act, nil
		}
	}
	if as.ContentExperienceActivityTypes.Contains(it.GetType()) {
		act, err = ContentExperienceActivity(l, act)
		if err == nil {
			return act, nil
		}
	}
	if as.GeoSocialEventsActivityTypes.Contains(it.GetType()) {
		act, err = GeoSocialEventsActivity(l, act)
		if err == nil {
			return act, nil
		}
	}
	if as.NotificationActivityTypes.Contains(it.GetType()) {
		act, err = NotificationActivity(l, act)
		if err == nil {
			return act, nil
		}
	}
	if as.QuestionActivityTypes.Contains(it.GetType()) {
		act, err = QuestionActivity(l, act)
		if err == nil {
			return act, nil
		}
	}
	if as.RelationshipManagementActivityTypes.Contains(it.GetType()) && act.Object.GetType() == as.RelationshipType {
		act, err = RelationshipManagementActivity(l, act)
		if err == nil {
			return act, errors.Annotatef(err, "%s activity processing failed", act.Type)
		}
	}
	if as.NegatingActivityTypes.Contains(it.GetType()) {
		act, err = NegatingActivity(l, act)
		if err == nil {
			return act, nil
		}
	}
	if as.OffersActivityTypes.Contains(it.GetType()) {
		act, err = OffersActivity(l, act)
		if err == nil {
			return act, nil
		}
	}
	return it, err
}


// ContentManagementActivity processes matching activities
func ContentManagementActivity(l s.Saver, act *Activity) (*Activity, error) {
	var err error
	if act.Object == nil {
		return act, errors.NotValidf("Missing object for Activity")
	}
	switch act.Type {
	case as.CreateType:
		// TODO(marius) Add function as.AttributedTo(it as.Item, auth as.Item)
		if a, err := ToActivity(act.Object); err == nil {
			// See https://www.w3.org/TR/ActivityPub/#create-activity-outbox
			// Copying the actor's IRI to the object's AttributedTo
			a.AttributedTo = act.Actor.GetLink()

			aRec := act.Recipients()
			// Copying the activity's recipients to the object's
			a.Audience = aRec
			// Copying the object's recipients to the activity's audience
			act.Audience = a.Recipients()

			act.Object = a
		} else if p, err := ToPerson(act.Object); err == nil {
			// See https://www.w3.org/TR/ActivityPub/#create-activity-outbox
			// Copying the actor's IRI to the object's AttributedTo
			p.AttributedTo = act.Actor.GetLink()

			aRec := act.Recipients()
			// Copying the activity's recipients to the object's
			p.Audience = aRec
			// Copying the object's recipients to the activity's audience
			act.Audience = p.Recipients()

			act.Object = p
		} else if o, err := ToObject(act.Object); err == nil {
			// See https://www.w3.org/TR/ActivityPub/#create-activity-outbox
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
		//l.errFn(logrus.Fields{"IRI": act.GetLink(), "type": act.Type}, "unable to save activity's object")
		return act, err
	}
	return act, err
}

// ReactionsActivity processes matching activities
func ReactionsActivity(l s.Saver, act *Activity) (*Activity, error) {
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
func CollectionManagementActivity(l s.Saver, act *Activity) (*Activity, error) {
	// TODO(marius):
	return nil, errors.Errorf("Not implemented")
}

// EventRSVPActivity processes matching activities
func EventRSVPActivity(l s.Saver, act *Activity) (*Activity, error) {
	// TODO(marius):
	return nil, errors.Errorf("Not implemented")
}

// GroupManagementActivity processes matching activities
func GroupManagementActivity(l s.Saver, act *Activity) (*Activity, error) {
	// TODO(marius):
	return nil, errors.Errorf("Not implemented")
}

// CollectionManagementActivity processes matching activities
func ContentExperienceActivity(l s.Saver, act *Activity) (*Activity, error) {
	// TODO(marius):
	return nil, errors.Errorf("Not implemented")
}

// GeoSocialEventsActivity processes matching activities
func GeoSocialEventsActivity(l s.Saver, act *Activity) (*Activity, error) {
	// TODO(marius):
	return nil, errors.Errorf("Not implemented")
}

// NotificationActivity processes matching activities
func NotificationActivity(l s.Saver, act *Activity) (*Activity, error) {
	// TODO(marius):
	return nil, errors.Errorf("Not implemented")
}

// QuestionActivity processes matching activities
func QuestionActivity(l s.Saver, act *Activity) (*Activity, error) {
	// TODO(marius):
	return nil, errors.Errorf("Not implemented")
}

// RelationshipManagementActivity processes matching activities
func RelationshipManagementActivity(l s.Saver, act *Activity) (*Activity, error) {
	// TODO(marius):
	return nil, errors.Errorf("Not implemented")
}

// NegatingActivity processes matching activities
func NegatingActivity(l s.Saver, act *Activity) (*Activity, error) {
	// TODO(marius):
	return nil, errors.Errorf("Not implemented")
}

// OffersActivity processes matching activities
func OffersActivity(l s.Saver, act *Activity) (*Activity, error) {
	// TODO(marius):
	return nil, errors.Errorf("Not implemented")
}
