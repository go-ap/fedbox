package validation

import (
	"context"
	xerrors "errors"
	"fmt"
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/client"
	"github.com/go-ap/errors"
	"github.com/go-ap/handlers"
	"github.com/go-ap/storage"
	"path"
	"strings"
)

type ClientActivityValidator interface {
	ValidateClientActivity(pub.Item, pub.IRI) error
	//ValidateClientObject(pub.Item) error
	ValidateClientActor(pub.Item) error
	//ValidateClientTarget(pub.Item) error
	//ValidateClientAudience(...pub.ItemCollection) error
}

type ServerActivityValidator interface {
	ValidateServerActivity(pub.Item, pub.IRI) error
	//ValidateServerObject(pub.Item) error
	ValidateServerActor(pub.Item) error
	//ValidateServerTarget(pub.Item) error
	//ValidateServerAudience(...pub.ItemCollection) error
}

// ActivityValidator is an interface used for validating activity objects.
type ActivityValidator interface {
	ClientActivityValidator
	ServerActivityValidator
}

//type AudienceValidator interface {
//	ValidateAudience(...pub.ItemCollection) error
//}

// ObjectValidator is an interface used for validating generic objects
type ObjectValidator interface {
	ValidateObject(pub.Item) error
}

// ActorValidator is an interface used for validating actor objects
type ActorValidator interface {
	ValidActor(pub.Item) error
}

// TargetValidator is an interface used for validating an object that is an activity's target
// TODO(marius): this seems to have a different semantic than the previous ones.
//  Ie, any object can be a target, but in the previous cases, the main validation mechanism is based on the Type.
//type TargetValidator interface {
//	ValidTarget(pub.Item) error
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
	baseIRI pub.IRI
	auth    *pub.Actor
	c       client.Client
	s       storage.Loader
}

func New(iri string, c client.Client, s storage.Loader) *genericValidator {
	return &genericValidator{
		baseIRI: pub.IRI(iri),
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

func (v genericValidator) ValidateServerActivity(a pub.Item, inbox pub.IRI) error {
	if !IsInbox(inbox) {
		return errors.NotValidf("Trying to validate a non inbox IRI %s", inbox)
	}
	//if v.auth.GetLink() == pub.PublicNS {
	//	return errors.Unauthorizedf("%s actor is not allowed posting to current inbox", v.auth.Name)
	//}
	if a == nil {
		return InvalidActivityActor("received nil activity")
	}
	if a.IsLink() {
		return v.ValidateLink(a.GetLink())
	}
	if !pub.ActivityTypes.Contains(a.GetType()) {
		return InvalidActivity("invalid type %s", a.GetType())
	}
	act, err := pub.ToActivity(a)
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

func IsOutbox(i pub.IRI) bool {
	return strings.ToLower(path.Base(i.String())) == strings.ToLower(string(handlers.Outbox))
}

func IsInbox(i pub.IRI) bool {
	return strings.ToLower(path.Base(i.String())) == strings.ToLower(string(handlers.Inbox))
}

// IRIBelongsToActor checks if the search iri represents any of the collections associated with the actor.
func IRIBelongsToActor(iri pub.IRI, actor *pub.Actor) bool {
	if actor == nil {
		return false
	}
	//p, _ := activitypub.ToPerson(actor)
	if actor.Inbox.GetLink().Equals(iri, false) {
		return true
	}
	if actor.Outbox.GetLink().Equals(iri, false) {
		return true
	}
	if actor.Endpoints != nil && actor.Endpoints.SharedInbox.GetLink().Equals(iri, false) {
		return true
	}
	// The following should not really come into question at any point.
	// This function should be used for checking inbox/outbox/sharedInbox IRIS
	if actor.Following.GetLink().Equals(iri, false) {
		return true
	}
	if actor.Followers.GetLink().Equals(iri, false) {
		return true
	}
	if actor.Replies.GetLink().Equals(iri, false) {
		return true
	}
	if actor.Liked.GetLink().Equals(iri, false) {
		return true
	}
	if actor.Shares.GetLink().Equals(iri, false) {
		return true
	}
	if actor.Likes.GetLink().Equals(iri, false) {
		return true
	}
	return false
}

var missingActor = new(MissingActorError)

func (v genericValidator) ValidateClientActivity(a pub.Item, outbox pub.IRI) error {
	if !IsOutbox(outbox) {
		return errors.NotValidf("Trying to validate a non outbox IRI %s", outbox)
	}
	if v.auth.GetLink() == pub.PublicNS {
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
	if !pub.ActivityTypes.Contains(a.GetType()) {
		return InvalidActivity("invalid type %s", a.GetType())
	}
	return pub.OnActivity(a, func(act *pub.Activity) error {
		if err := v.ValidateClientActor(act.Actor); err != nil {
			if missingActor.Is(err) && v.auth != nil {
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
		var err error
		if pub.ContentManagementActivityTypes.Contains(act.GetType()) && act.Object.GetType() != pub.RelationshipType {
			err = ValidateClientContentManagementActivity(v.s, act)
		} else if pub.CollectionManagementActivityTypes.Contains(act.GetType()) {
			err = ValidateClientCollectionManagementActivity(v.s, act)
		} else if pub.ReactionsActivityTypes.Contains(act.GetType()) {
			err = ValidateClientReactionsActivity(v.s, act)
		} else if pub.EventRSVPActivityTypes.Contains(act.GetType()) {
			err = ValidateClientEventRSVPActivity(v.s, act)
		} else if pub.GroupManagementActivityTypes.Contains(act.GetType()) {
			err = ValidateClientGroupManagementActivity(v.s, act)
		} else if pub.ContentExperienceActivityTypes.Contains(act.GetType()) {
			err = ValidateClientContentExperienceActivity(v.s, act)
		} else if pub.GeoSocialEventsActivityTypes.Contains(act.GetType()) {
			err = ValidateClientGeoSocialEventsActivity(v.s, act)
		} else if pub.NotificationActivityTypes.Contains(act.GetType()) {
			err = ValidateClientNotificationActivity(v.s, act)
		} else if pub.QuestionActivityTypes.Contains(act.GetType()) {
			err = ValidateClientQuestionActivity(v.s, act)
		} else if pub.RelationshipManagementActivityTypes.Contains(act.GetType()) {
			err = ValidateClientRelationshipManagementActivity(v.s, act)
		} else if pub.NegatingActivityTypes.Contains(act.GetType()) {
			err = ValidateClientNegatingActivity(v.s, act)
		} else if pub.OffersActivityTypes.Contains(act.GetType()) {
			err = ValidateClientOffersActivity(v.s, act)
		}
		return err
	})
}

// ValidateClientContentManagementActivity
func ValidateClientContentManagementActivity(l storage.Loader, act *pub.Activity) error {
	if act.Object == nil {
		return errors.NotValidf("nil object for %s activity", act.Type)
	}
	ob := act.Object
	switch act.Type {
	case pub.UpdateType:
		if pub.ActivityTypes.Contains(ob.GetType()) {
			return errors.Newf("trying to update an immutable activity")
		}
		fallthrough
	case pub.DeleteType:
		if len(ob.GetLink()) == 0 {
			return errors.Newf("invalid object id for %s activity", act.Type)
		}
		typ := ob.GetType()

		var (
			found pub.Item
			err   error
			cnt   uint
		)
		if pub.ActorTypes.Contains(typ) {
			found, cnt, err = l.LoadActors(ob)
		}
		if pub.ObjectTypes.Contains(typ) {
			found, cnt, err = l.LoadObjects(ob)
		}
		if err != nil {
			return errors.Annotatef(err, "failed to load object from storage")
		}
		if cnt == 0 {
			return errors.NotFoundf("unable to find %s %s in storage", ob.GetType(), ob.GetLink())
		}
		if found == nil {
			return errors.NotFoundf("found nil object in storage")
		}
	case pub.CreateType:
	default:
	}

	return nil
}

// ValidateClientCollectionManagementActivity
func ValidateClientCollectionManagementActivity(l storage.Loader, act *pub.Activity) error {
	return nil
}

// ValidateClientReactionsActivity
func ValidateClientReactionsActivity(l storage.Loader, act *pub.Activity) error {
	return nil
}

// ValidateClientEventRSVPActivity
func ValidateClientEventRSVPActivity(l storage.Loader, act *pub.Activity) error {
	return nil
}

// ValidateClientGroupManagementActivity
func ValidateClientGroupManagementActivity(l storage.Loader, act *pub.Activity) error {
	return nil
}

// ValidateClientContentExperienceActivity
func ValidateClientContentExperienceActivity(l storage.Loader, act *pub.Activity) error {
	return nil
}

// ValidateClientGeoSocialEventsActivity
func ValidateClientGeoSocialEventsActivity(l storage.Loader, act *pub.Activity) error {
	return nil
}

// ValidateClientNotificationActivity
func ValidateClientNotificationActivity(l storage.Loader, act *pub.Activity) error {
	return nil
}

// ValidateClientQuestionActivity
func ValidateClientQuestionActivity(l storage.Loader, act *pub.Activity) error {
	return nil
}

// ValidateClientRelationshipManagementActivity
func ValidateClientRelationshipManagementActivity(l storage.Loader, act *pub.Activity) error {
	switch act.Type {
	case pub.FollowType:
		_, cnt, _ := l.LoadActivities(storage.FilterItem(act))
		if cnt > 0 {
			return errors.Newf("%s already exists for this actor/object pair", act.Type)
		}
	case pub.AddType:
	case pub.BlockType:
	case pub.CreateType:
	case pub.DeleteType:
	case pub.IgnoreType:
	case pub.InviteType:
	case pub.AcceptType:
		fallthrough
	case pub.RejectType:
		// TODO(marius): either the actor or the object needs to be local for this action to be valid
		//   in the case of C2S... the actor needs to be local
		//   in the case of S2S... the object needs to be local
		// TODO(marius): Object needs to be a valid Follow activity
	default:
	}
	return nil
}

// ValidateClientNegatingActivity
func ValidateClientNegatingActivity(l storage.Loader, act *pub.Activity) error {
	return nil
}

// ValidateClientOffersActivity
func ValidateClientOffersActivity(l storage.Loader, act *pub.Activity) error {
	return nil
}

// IsLocalIRI shows if the received IRI belongs to the current instance
// TODO(marius): make this not be true always
func (v genericValidator) IsLocalIRI(i pub.IRI) bool {
	return i.Contains(v.baseIRI, false)
}

func (v genericValidator) ValidateLink(i pub.IRI) error {
	if i.Equals(pub.PublicNS, false) {
		return InvalidActivityActor("Public namespace is not a local actor")
	}
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

func (v genericValidator) ValidateClientActor(a pub.Item) error {
	if a == nil {
		return MissingActivityActor("")
	}
	if err := v.validateLocalIRI(a.GetLink()); err != nil {
		return InvalidActivityActor("%s is not local", a.GetLink())
	}
	return v.ValidateActor(a)
}

func (v genericValidator) ValidateServerActor(a pub.Item) error {
	return v.ValidateActor(a)
}

func (v genericValidator) ValidateActor(a pub.Item) error {
	if a == nil {
		return InvalidActivityActor("is nil")
	}
	if a.IsLink() {
		return v.ValidateLink(a.GetLink())
	}
	if !pub.ActorTypes.Contains(a.GetType()) {
		return InvalidActivityActor("invalid type %s", a.GetType())
	}
	if v.auth != nil {
		if v.auth.GetLink().String() == a.GetLink().String() {
			return InvalidActivityActor("current activity's actor doesn't match the authenticated one")
		}
	}
	return nil
}

func (v genericValidator) ValidateClientObject(o pub.Item) error {
	return v.ValidateObject(o)
}

func (v genericValidator) ValidateServerObject(o pub.Item) error {
	return v.ValidateObject(o)
}

func (v genericValidator) ValidateObject(o pub.Item) error {
	if o == nil {
		return InvalidActivityObject("is nil")
	}
	if o.IsLink() {
		return v.ValidateLink(o.GetLink())
	}
	if !(pub.ObjectTypes.Contains(o.GetType()) || pub.ActorTypes.Contains(o.GetType())) {
		return InvalidActivityObject("invalid type %s", o.GetType())
	}
	return nil
}

func (v genericValidator) ValidateTarget(t pub.Item) error {
	if t == nil {
		return InvalidActivityObject("is nil")
	}
	if t.IsLink() {
		return v.ValidateLink(t.GetLink())
	}
	if !(pub.ObjectTypes.Contains(t.GetType()) || pub.ActorTypes.Contains(t.GetType()) || pub.ActivityTypes.Contains(t.GetType())) {
		return InvalidActivityObject("invalid type %s", t.GetType())
	}
	return nil
}

func (v genericValidator) ValidateAudience(audience ...pub.ItemCollection) error {
	for _, elem := range audience {
		for _, iri := range elem {
			if err := v.validateLocalIRI(iri.GetLink()); err == nil {
				return nil
			}
			if iri.GetLink() == pub.PublicNS {
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

func (v *genericValidator) SetActor(p *pub.Actor) {
	v.auth = p
}

func (v genericValidator) validateLocalIRI(i pub.IRI) error {
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
