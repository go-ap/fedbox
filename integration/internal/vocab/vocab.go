package vocab

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	vocab "github.com/go-ap/activitypub"
)

type (
	iri  = vocab.IRI
	iris = vocab.IRIs
	t    = vocab.ActivityVocabularyType
	ts   = vocab.ActivityVocabularyTypes
	ic   = vocab.ItemCollection
	i    = vocab.Item
	ep   = vocab.Endpoints
	o    = vocab.Object
	a    = vocab.Actor
	aa   = vocab.Activity
	ai   = vocab.IntransitiveActivity
	q    = vocab.Question
	oc   = vocab.OrderedCollection
	c    = vocab.Collection

	InitFn = any
)

var (
	EN = vocab.DefaultNaturalLanguage[string]
)

func NL[T ~string](content T) vocab.NaturalLanguageValues {
	return vocab.NaturalLanguageValuesNew(vocab.RefValue(vocab.NilLangRef, content))
}

func HasAttributedTo(i ...iri) func(*o) error {
	is := iris(i)
	return func(ob *o) error {
		ob.AttributedTo = is.Collection().Normalize()
		return nil
	}
}

func HasContext(i ...iri) func(*o) error {
	is := iris(i)
	return func(ob *o) error {
		ob.Context = is.Collection().Normalize()
		return nil
	}
}

func HasAudience(i ...iri) func(*o) error {
	aud := iris(i)
	return func(ob *o) error {
		ob.Audience = aud.Collection()
		return nil
	}
}

func HasGenerator(i ...iri) func(*o) error {
	is := iris(i)
	return func(ob *o) error {
		ob.Generator = is.Collection().Normalize()
		return nil
	}
}

func HasURL(i ...iri) func(*o) error {
	is := iris(i)
	return func(ob *o) error {
		ob.URL = is.Collection().Normalize()
		return nil
	}
}

func HasStream(i iri) func(*a) error {
	return func(ob *a) error {
		if ob.Streams == nil {
			ob.Streams = make(ic, 0)
		}
		return ob.Streams.Append(i)
	}
}

func HasPublicKey(k crypto.PublicKey) func(*a) error {
	pubEnc, _ := x509.MarshalPKIXPublicKey(k)
	pubEncoded := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubEnc})
	return func(ob *a) error {
		ob.PublicKey = vocab.PublicKey{
			ID:           vocab.IRI(fmt.Sprintf("%s#main", ob.ID)),
			Owner:        ob.ID,
			PublicKeyPem: string(pubEncoded),
		}
		return nil
	}
}

func HasTag(t i) func(*o) error {
	return func(ob *o) error {
		return ob.Tag.Append(t)
	}
}

func HasCC(i iri) func(*o) error {
	return func(ob *o) error {
		return ob.CC.Append(i)
	}
}

func HasTo(i iri) func(*o) error {
	return func(ob *o) error {
		return ob.To.Append(i)
	}
}

func nlv[T ~string | vocab.NaturalLanguageValues](c T) vocab.NaturalLanguageValues {
	var result vocab.NaturalLanguageValues
	switch v := any(c).(type) {
	case string:
		if v != "" {
			result = vocab.DefaultNaturalLanguage(v)
		}
	case []byte:
		result = vocab.DefaultNaturalLanguage(string(v))
	case vocab.NaturalLanguageValues:
		result = v
	}
	return result
}

func HasName[T ~string | vocab.NaturalLanguageValues](c T) func(*o) error {
	v := nlv(c)
	return func(ob *o) error {
		ob.Name = v
		return nil
	}
}

func HasSummary[T ~string | vocab.NaturalLanguageValues](c T) func(*o) error {
	v := nlv(c)
	return func(ob *o) error {
		ob.Summary = v
		return nil
	}
}

func HasSource(c string, mt string) func(*o) error {
	return func(ob *o) error {
		ob.Source.Content = NL(c)
		ob.Source.MediaType = vocab.MimeType(mt)
		return nil
	}
}

func HasContent[T ~string | vocab.NaturalLanguageValues](c T) func(*o) error {
	v := nlv(c)
	return func(ob *o) error {
		ob.Content = v
		return nil
	}
}

func HasMediaType(m string) func(*o) error {
	return func(ob *o) error {
		ob.MediaType = vocab.MimeType(m)
		return nil
	}
}

func HasPublished[T string | time.Time](s T) func(*o) error {
	var p time.Time
	switch ss := any(s).(type) {
	case string:
		p, _ = time.Parse(time.RFC3339Nano, ss)
	case time.Time:
		p = ss
	}
	return func(ob *o) error {
		ob.Published = p
		return nil
	}
}

func HasUpdated[T string | time.Time](s T) func(*o) error {
	var u time.Time
	switch ss := any(s).(type) {
	case string:
		u, _ = time.Parse(time.RFC3339Nano, ss)
	case time.Time:
		u = ss
	}
	return func(ob *o) error {
		ob.Updated = u
		return nil
	}
}

func HasPreferredUsername[T ~string | vocab.NaturalLanguageValues](c T) func(*a) error {
	v := nlv(c)
	return func(act *a) error {
		act.PreferredUsername = v
		return nil
	}
}

func HasProxyURL(i iri) func(*a) error {
	return func(act *a) error {
		if act.Endpoints == nil {
			act.Endpoints = new(ep)
		}
		act.Endpoints.ProxyURL = i
		return nil
	}
}

func HasSharedInbox(i iri) func(*a) error {
	return func(act *a) error {
		if act.Endpoints == nil {
			act.Endpoints = new(ep)
		}
		act.Endpoints.SharedInbox = i
		return nil
	}
}

func HasReplies(ob *o) error {
	ob.Replies = vocab.Replies.IRI(ob.ID)
	return nil
}

func HasLikes(ob *o) error {
	ob.Likes = vocab.Likes.IRI(ob.ID)
	return nil
}

func HasLiked(act *a) error {
	act.Liked = vocab.Liked.IRI(act.ID)
	return nil
}

func HasShares(ob *o) error {
	ob.Shares = vocab.Shares.IRI(ob.ID)
	return nil
}

func HasFollowing(act *a) error {
	act.Following = vocab.Following.IRI(act.ID)
	return nil
}

func HasFollowers(act *a) error {
	act.Followers = vocab.Followers.IRI(act.ID)
	return nil
}

func HasID(i iri) func(*o) error {
	return func(ob *o) error {
		ob.ID = i
		return nil
	}
}

func HasType(t ...t) func(*o) error {
	return func(ob *o) error {
		if len(t) == 1 {
			ob.Type = t[0]
		} else {
			ob.Type = ts(t)
		}
		return nil
	}
}

func HasInReplyTo(c ...i) func(*o) error {
	return func(ob *o) error {
		ob.InReplyTo = ic(c).Normalize()
		return nil
	}
}

func HasActor(a ...i) func(*ai) error {
	return func(act *ai) error {
		act.Actor = ic(a).Normalize()
		return nil
	}
}

func HasObject(o ...i) func(*aa) error {
	return func(act *aa) error {
		act.Object = ic(o).Normalize()
		return nil
	}
}

func HasOrigin(o ...i) func(*aa) error {
	return func(act *aa) error {
		act.Origin = ic(o).Normalize()
		return nil
	}
}

func HasTarget(t ...i) func(*aa) error {
	return func(act *aa) error {
		act.Target = ic(t).Normalize()
		return nil
	}
}

func Object(initFn ...InitFn) *o {
	ob := o{}
	for _, maybeFn := range initFn {
		switch fn := maybeFn.(type) {
		case vocab.IRI:
			ob.ID = fn
		case func(*o) error:
			_ = vocab.OnObject(&ob, fn)
		}
	}
	return &ob
}

func Activity(initFn ...InitFn) *aa {
	act := new(aa)
	for _, maybeFn := range initFn {
		switch fn := maybeFn.(type) {
		case func(*o) error:
			_ = vocab.OnObject(act, fn)
		case func(*ai) error:
			_ = vocab.OnIntransitiveActivity(act, fn)
		case func(*aa) error:
			_ = vocab.OnActivity(act, fn)
		}
	}
	return act
}

func Question(initFn ...InitFn) *q {
	act := new(q)
	for _, maybeFn := range initFn {
		switch fn := maybeFn.(type) {
		case func(*o) error:
			_ = vocab.OnObject(act, fn)
		case func(*ai) error:
			_ = vocab.OnIntransitiveActivity(act, fn)
		case func(*q) error:
			_ = vocab.OnQuestion(act, fn)
		}
	}
	return act
}

func OrderedCollection(initFn ...InitFn) *oc {
	ob := oc{}
	for _, maybeFn := range initFn {
		switch fn := maybeFn.(type) {
		case vocab.IRI:
			ob.ID = fn
		case func(*o) error:
			_ = vocab.OnObject(&ob, fn)
		case func(*oc) error:
			_ = vocab.OnOrderedCollection(&ob, fn)
		}
	}
	return &ob
}

func Collection(initFn ...InitFn) *c {
	ob := c{}
	for _, maybeFn := range initFn {
		switch fn := maybeFn.(type) {
		case vocab.IRI:
			ob.ID = fn
		case func(*o) error:
			_ = vocab.OnObject(&ob, fn)
		case func(*c) error:
			_ = vocab.OnCollection(&ob, fn)
		}
	}
	return &ob
}

func AnyOf(o ...i) func(*q) error {
	return func(act *q) error {
		act.AnyOf = ic(o).Normalize()
		return nil
	}
}

func OneOf(o ...i) func(*q) error {
	return func(act *q) error {
		act.OneOf = ic(o).Normalize()
		return nil
	}
}

func IntransitiveActivity(initFn ...InitFn) *ai {
	act := new(ai)
	for _, maybeFn := range initFn {
		switch fn := maybeFn.(type) {
		case func(*o) error:
			_ = vocab.OnObject(act, fn)
		case func(*ai) error:
			_ = vocab.OnIntransitiveActivity(act, fn)
		}
	}
	return act
}

func HasAuthEp(i iri) func(*a) error {
	return func(act *a) error {
		if act.Endpoints == nil {
			act.Endpoints = new(ep)
		}
		act.Endpoints.OauthAuthorizationEndpoint = i
		return nil
	}
}

func HasTokenEp(i iri) func(*a) error {
	return func(act *a) error {
		if act.Endpoints == nil {
			act.Endpoints = new(ep)
		}
		act.Endpoints.OauthTokenEndpoint = i
		return nil
	}
}

func Actor(initFn ...InitFn) *a {
	act := a{}
	for _, maybeFn := range initFn {
		switch fn := maybeFn.(type) {
		case func(*vocab.Object) error:
			_ = vocab.OnObject(&act, fn)
		case func(*vocab.Actor) error:
			_ = vocab.OnActor(&act, fn)
		}
	}
	if act.Inbox == nil {
		act.Inbox = vocab.Inbox.IRI(act)
	}
	if act.Outbox == nil {
		act.Outbox = vocab.Outbox.IRI(act)
	}
	return &act
}
