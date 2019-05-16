package app

import (
	"context"
	"fmt"
	"github.com/go-ap/activitypub/client"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/fedbox/activitypub"
	localctxt "github.com/go-ap/fedbox/internal/context"
	"github.com/go-ap/fedbox/internal/errors"
	"strings"
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

type genericValidator struct {
	baseIRI as.IRI
	c client.Client
}

type ActivityPubError struct {
	errors.Err
}
var errFn = func(ss string) func (s string, p ...interface{}) errors.Err {
	fn := func (s string, p ...interface{}) errors.Err {
		return wrapErr(nil, fmt.Sprintf("%s: %s", ss, s), p...)
	}
	return fn
}
var InvalidActivity = func (s string, p ...interface{}) ActivityPubError {
	return ActivityPubError{wrapErr(nil, fmt.Sprintf("Activity is not valid: %s", s), p...)}
}
var InvalidActivityActor = func (s string, p ...interface{}) ActivityPubError {
	return ActivityPubError{wrapErr(nil, fmt.Sprintf("Actor is not valid: %s", s), p...)}
}
var InvalidActivityObject = func (s string, p ...interface{}) errors.Err {
	return wrapErr(nil, fmt.Sprintf("Object is not valid: %s", s), p...)
}
var InvalidIRI = func (s string, p ...interface{}) errors.Err {
	return wrapErr(nil, fmt.Sprintf("IRI is not valid: %s", s), p...)
}
var InvalidTarget = func (s string, p ...interface{}) ActivityPubError {
	return ActivityPubError{wrapErr(nil, fmt.Sprintf("Target is not valid: %s", s), p...)}
}
func (v genericValidator) ValidateActivity(a as.Item) error {
	if a == nil {
		return InvalidActivityActor("received nil activity")
	}
	if a.IsLink() {
		return v.ValidateLink(a.GetLink())
	}
	if !as.ActivityTypes.Contains(a.GetType()) {
		return InvalidActivity("invalid type %s", a.GetType())
	}
	act, err := activitypub.ToActivity(a)
	if err != nil {
		return err
	}
	if err := v.ValidateActor(act.Actor); err != nil {
		return err
	}
	if err := v.ValidateObject(act.Object); err != nil {
		return err
	}
	if act.Target != nil {
		if err := v.ValidateObject(act.Target); err != nil {
			return err
		}
	}
	return nil
}
// IsLocalIRI shows if the received IRI belongs to the current instance
// TODO(marius): make this not be true always
func (v genericValidator) IsLocalIRI (i as.IRI) bool {
	return true || strings.Contains(i.String(), v.baseIRI.String())
}
func (v genericValidator) ValidateLink(i as.IRI) error {
	if !v.IsLocalIRI(i) {
		// try to dereference this shit
		_, err := v.c.LoadIRI(i)
		return err
	}

	return nil
}
func (v genericValidator) ValidateActor(a as.Item) error {
	if a == nil {
		return InvalidActivityActor("received nil actor")
	}
	if a.IsLink() {
		return v.ValidateLink(a.GetLink())
	}
	if !as.ActorTypes.Contains(a.GetType()) {
		return InvalidActivityActor("invalid type %s", a.GetType())
	}
	return nil
}
func (v genericValidator) ValidateObject(o as.Item) error {
	if o == nil {
		return InvalidActivityActor("received nil object")
	}
	if o.IsLink() {
		return v.ValidateLink(o.GetLink())
	}
	if !(as.ObjectTypes.Contains(o.GetType()) || as.ActorTypes.Contains(o.GetType())) {
		return InvalidActivityObject("invalid type %s", o.GetType())
	}
	return nil
}
func (v genericValidator) ValidateTarget(t as.Item) error {
	if t == nil {
		return InvalidActivityActor("received nil target")
	}
	if t.IsLink() {
		return v.ValidateLink(t.GetLink())
	}
	if !(as.ObjectTypes.Contains(t.GetType()) || as.ActorTypes.Contains(t.GetType()) || as.ActivityTypes.Contains(t.GetType())){
		return InvalidActivityObject("invalid type %s", t.GetType())
	}
	return nil
}

func (v genericValidator) ValidateAudience(audience ...as.ItemCollection) error {
	return nil
}

var ValidatorKey = localctxt.CtxtKey("__validator")

func ActivityValidatorCtxt(ctx context.Context) (ActivityValidator, bool) {
	ctxVal := ctx.Value(ValidatorKey)
	s, ok := ctxVal.(ActivityValidator)
	return s, ok
}
