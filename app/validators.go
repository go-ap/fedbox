package app

import (
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/fedbox/internal/errors"
)

// ActivityValidator is an interface used for validating activity objects.
type ActivityValidator interface {
	ValidateActivity(as.Item) error
	ValidateObject(as.Item) error
	ValidateActor(as.Item) error
	ValidateTarget(as.Item) error
	ValidateAudience(...as.ItemCollection) error
}

//type AudienceValidator interface {
//	ValidateAudience(...as.ItemCollection) error
//}
// ObjectValidator is an interface used for validating generic objects
//type ObjectValidator interface {
//	ValidateObject(as.Item) error
//}

// ActorValidator is an interface used for validating actor objects
//type ActorValidator interface {
//	ValidActor(as.Item) error
//}

// TargetValidator is an interface used for validating an object that is an activity's target
// TODO(marius): this seems to have a different semantic than the previous ones.
//  Ie, any object can be a target, but in the previous cases, the main validation mechanism is based on the Type.
//type TargetValidator interface {
//	ValidTarget(as.Item) error
//}

type invalidActivity struct {
	errors.Err
}

type genericValidator struct{}

type ActivityPubError errors.Err

var InvalidActivity = wrapErr(nil, "Activity is not valid")
var InvalidActivityActor = wrapErr(nil,"Actor is not valid")
var InvalidActivityObject = wrapErr(nil,"Object is not valid")
var InvalidTarget = wrapErr(nil,"Target is not valid")

func (v genericValidator) ValidateActivity(a as.Item) error {
	if !as.ValidActivityType(a.GetType()) {
		return InvalidActivity
	}
	act, err := as.ToActivity(a)
	if err != nil {
		return errors.Annotatef(err, "")
	}
	if err := v.ValidateActor(act.Actor); err != nil {
		return errors.Annotatef(err, "")
	}
	if err := v.ValidateObject(act.Object); err != nil {
		return errors.Annotatef(err, "")
	}
	if act.Target != nil {
		if err := v.ValidateObject(act.Target); err != nil {
			return errors.Annotatef(err, "")
		}
	}
	return nil
}
func (v genericValidator) ValidateActor(a as.Item) error {
	if !as.ValidActorType(a.GetType()) {
		return InvalidActivityActor
	}
	return nil
}
func (v genericValidator) ValidateObject(a as.Item) error {
	if !as.ValidObjectType(a.GetType()) {
		return InvalidActivityObject
	}
	return nil
}
func (v genericValidator) ValidateTarget(a as.Item) error {
	if !as.ValidObjectType(a.GetType()) {
		return InvalidActivityObject
	}
	return nil
}
func (v genericValidator) ValidateAudience(a as.Item) error {
	return nil
}
