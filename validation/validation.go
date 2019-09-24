package validation

import (
	"context"
	xerrors "errors"
	"fmt"
	"github.com/go-ap/activitypub/client"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	"github.com/go-ap/handlers"
	"github.com/go-ap/storage"
	"path"
	"strings"
)

type ClientActivityValidator interface {
	ValidateClientActivity(as.Item, as.IRI) error
	//ValidateClientObject(as.Item) error
	ValidateClientActor(as.Item) error
	//ValidateClientTarget(as.Item) error
	//ValidateClientAudience(...as.ItemCollection) error
}

type ServerActivityValidator interface {
	ValidateServerActivity(as.Item, as.IRI) error
	//ValidateServerObject(as.Item) error
	ValidateServerActor(as.Item) error
	//ValidateServerTarget(as.Item) error
	//ValidateServerAudience(...as.ItemCollection) error
}

// ActivityValidator is an interface used for validating activity objects.
type ActivityValidator interface {
	ClientActivityValidator
	ServerActivityValidator
}

//type AudienceValidator interface {
//	ValidateAudience(...as.ItemCollection) error
//}

// ObjectValidator is an interface used for validating generic objects
type ObjectValidator interface {
	ValidateObject(as.Item) error
}

// ActorValidator is an interface used for validating actor objects
type ActorValidator interface {
	ValidActor(as.Item) error
}

// TargetValidator is an interface used for validating an object that is an activity's target
// TODO(marius): this seems to have a different semantic than the previous ones.
//  Ie, any object can be a target, but in the previous cases, the main validation mechanism is based on the Type.
//type TargetValidator interface {
//	ValidTarget(as.Item) error
//}

func wrapErr(err error, s string, args ...interface{}) errors.Err {
	e := errors.Annotatef(err, s, args...)
	asErr := errors.Err{}
	xerrors.As(e, &asErr)
	return asErr
}

type invalidActivity struct {
	errors.Err
}

type genericValidator struct {
	baseIRI as.IRI
	auth    *auth.Person
	c       client.Client
	s       storage.Loader
}

func New(iri string, c client.Client, s storage.Loader) *genericValidator {
	return &genericValidator{
		baseIRI: as.IRI(iri),
		c:       c,
		s:       s,
	}
}

type ActivityPubError struct {
	errors.Err
}

type MissingActorError struct {
	errors.Err
}

var errFn = func(ss string) func(s string, p ...interface{}) errors.Err {
	fn := func(s string, p ...interface{}) errors.Err {
		return wrapErr(nil, fmt.Sprintf("%s: %s", ss, s), p...)
	}
	return fn
}
var InvalidActivity = func(s string, p ...interface{}) ActivityPubError {
	return ActivityPubError{wrapErr(nil, fmt.Sprintf("Activity is not valid: %s", s), p...)}
}
var MissingActivityActor = func(s string, p ...interface{}) MissingActorError {
	return MissingActorError{wrapErr(nil, fmt.Sprintf("Missing actor %s", s), p...)}
}
var InvalidActivityActor = func(s string, p ...interface{}) ActivityPubError {
	return ActivityPubError{wrapErr(nil, fmt.Sprintf("Actor is not valid: %s", s), p...)}
}
var InvalidActivityObject = func(s string, p ...interface{}) errors.Err {
	return wrapErr(nil, fmt.Sprintf("Object is not valid: %s", s), p...)
}
var InvalidIRI = func(s string, p ...interface{}) errors.Err {
	return wrapErr(nil, fmt.Sprintf("IRI is not valid: %s", s), p...)
}
var InvalidTarget = func(s string, p ...interface{}) ActivityPubError {
	return ActivityPubError{wrapErr(nil, fmt.Sprintf("Target is not valid: %s", s), p...)}
}

func (m *MissingActorError) Is(e error) bool {
	_, okp := e.(*MissingActorError)
	_, oks := e.(MissingActorError)
	return okp || oks
}

func (v genericValidator) ValidateServerActivity(a as.Item, inbox as.IRI) error {
	if !IsInbox(inbox) {
		return errors.NotValidf("Trying to validate a non inbox IRI %s", inbox)
	}
	if v.auth.GetLink() == as.PublicNS {
		return errors.Unauthorizedf("%s actor is not allowed posting to current inbox", v.auth.Name)
	}
	if a == nil {
		return InvalidActivityActor("received nil activity")
	}
	if a.IsLink() {
		return v.ValidateLink(a.GetLink())
	}
	if !as.ActivityTypes.Contains(a.GetType()) {
		return InvalidActivity("invalid type %s", a.GetType())
	}
	act, err := as.ToActivity(a)
	if err != nil {
		return err
	}
	if err := v.ValidateServerActor(act.Actor); err != nil {
		if (&MissingActorError{}).Is(err) && v.auth != nil {
			act.Actor = v.auth
		} else {
			return err
		}
	}
	if err := v.ValidateServerObject(act.Object); err != nil {
		return err
	}
	if act.Target != nil {
		if err := v.ValidateServerObject(act.Target); err != nil {
			return err
		}
	}
	return nil
}

func IsOutbox(i as.IRI) bool {
	return strings.ToLower(path.Base(i.String())) == strings.ToLower(string(handlers.Outbox))
}

func IsInbox(i as.IRI) bool {
	return strings.ToLower(path.Base(i.String())) == strings.ToLower(string(handlers.Inbox))
}

// IRIBelongsToActor checks if the search iri represents any of the collections associated with the actor.
func IRIBelongsToActor(iri as.IRI, actor *auth.Person) bool {
	if actor == nil {
		return false
	}
	//p, _ := activitypub.ToPerson(actor)
	if actor.Inbox == iri {
		return true
	}
	if actor.Outbox == iri {
		return true
	}
	if actor.Endpoints != nil && actor.Endpoints.SharedInbox == iri {
		return true
	}
	// The following should not really come into question at any point.
	// This function should be used for checking inbox/outbox/sharedInbox IRIS
	if actor.Following == iri {
		return true
	}
	if actor.Followers == iri {
		return true
	}
	if actor.Replies == iri {
		return true
	}
	if actor.Liked == iri {
		return true
	}
	if actor.Shares == iri {
		return true
	}
	if actor.Likes == iri {
		return true
	}
	return false
}

func (v genericValidator) ValidateClientActivity(a as.Item, outbox as.IRI) error {
	if !IsOutbox(outbox) {
		return errors.NotValidf("Trying to validate a non outbox IRI %s", outbox)
	}
	if v.auth.GetLink() == as.PublicNS {
		return errors.Unauthorizedf("%s actor is not allowed posting to current outbox", v.auth.Name)
	}
	if !IRIBelongsToActor(outbox, v.auth) {
		return errors.Unauthorizedf("%s actor does not own the current outbox", v.auth.Name)
	}
	if a == nil {
		return InvalidActivityActor("received nil activity")
	}
	if a.IsLink() {
		return v.ValidateLink(a.GetLink())
	}
	if !as.ActivityTypes.Contains(a.GetType()) {
		return InvalidActivity("invalid type %s", a.GetType())
	}
	act, err := as.ToActivity(a)
	if err != nil {
		return err
	}
	if err := v.ValidateClientActor(act.Actor); err != nil {
		if (&MissingActorError{}).Is(err) && v.auth != nil {
			act.Actor = v.auth
		} else {
			return err
		}
	}
	if err := v.ValidateClientObject(act.Object); err != nil {
		return err
	}
	if act.Target != nil {
		if err := v.ValidateClientObject(act.Target); err != nil {
			return err
		}
	}
	return nil
}

// IsLocalIRI shows if the received IRI belongs to the current instance
// TODO(marius): make this not be true always
func (v genericValidator) IsLocalIRI(i as.IRI) bool {
	return strings.Contains(i.String(), v.baseIRI.String())
}

func (v genericValidator) ValidateLink(i as.IRI) error {
	if !v.IsLocalIRI(i) {
		// try to dereference this shit
		_, err := v.c.LoadIRI(i)
		return err
	} else {
		actors, cnt, err := v.s.LoadActors(i)
		if err != nil {
			return err
		}
		if cnt == 0 || len(actors) != int(cnt) {
			return InvalidActivityActor("%s could not be found locally", i)
		}
	}

	return nil
}

func (v genericValidator) ValidateClientActor(a as.Item) error {
	if a == nil {
		return MissingActivityActor("")
	}
	if err := v.validateLocalIRI(a.GetLink()); err != nil {
		return InvalidActivityActor("%s is not local", a.GetLink())
	}
	return v.ValidateActor(a)
}

func (v genericValidator) ValidateServerActor(a as.Item) error {
	return v.ValidateActor(a)
}

func (v genericValidator) ValidateActor(a as.Item) error {
	if a == nil {
		return InvalidActivityActor("is nil")
	}
	if a.IsLink() {
		return v.ValidateLink(a.GetLink())
	}
	if !as.ActorTypes.Contains(a.GetType()) {
		return InvalidActivityActor("invalid type %s", a.GetType())
	}
	if v.auth != nil {
		if v.auth.GetLink().String() == a.GetLink().String() {
			return InvalidActivityActor("current activity's actor doesn't match the authenticated one")
		}
	}
	return nil
}

func (v genericValidator) ValidateClientObject(o as.Item) error {
	return v.ValidateObject(o)
}

func (v genericValidator) ValidateServerObject(o as.Item) error {
	return v.ValidateObject(o)
}

func (v genericValidator) ValidateObject(o as.Item) error {
	if o == nil {
		return InvalidActivityObject("is nil")
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
		return InvalidActivityObject("is nil")
	}
	if t.IsLink() {
		return v.ValidateLink(t.GetLink())
	}
	if !(as.ObjectTypes.Contains(t.GetType()) || as.ActorTypes.Contains(t.GetType()) || as.ActivityTypes.Contains(t.GetType())) {
		return InvalidActivityObject("invalid type %s", t.GetType())
	}
	return nil
}

func (v genericValidator) ValidateAudience(audience ...as.ItemCollection) error {
	for _, elem := range audience {
		for _, iri := range elem {
			if err := v.validateLocalIRI(iri.GetLink()); err == nil {
				return nil
			}
			if iri.GetLink() == as.PublicNS {
				return nil
			}
		}
	}
	return errors.Newf("None of the audience elements is local")
}

type CtxtKey string

var ValidatorKey = CtxtKey("__validator")

func FromContext(ctx context.Context) (*genericValidator, bool) {
	ctxVal := ctx.Value(ValidatorKey)
	s, ok := ctxVal.(*genericValidator)
	return s, ok
}

func (v *genericValidator) SetActor(p *auth.Person) {
	v.auth = p
}

func (v genericValidator) validateLocalIRI(i as.IRI) error {
	u1, err := i.URL()
	if err != nil {
		return err
	}
	u2, err := v.baseIRI.URL()
	if err != nil {
		return err
	}
	if u1.Host != u2.Host {
		return errors.Newf("%s is not a local IRI", i)
	}
	return nil
}
