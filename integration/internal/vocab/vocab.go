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
	iri = vocab.IRI
	t   = vocab.ActivityVocabularyType
	ic  = vocab.ItemCollection
	i   = vocab.Item
	ep  = vocab.Endpoints
	o   = vocab.Object
	a   = vocab.Actor

	InitFn = any
)

var (
	EN = vocab.DefaultNaturalLanguage[string]
)

func NL[T ~string](content T) vocab.NaturalLanguageValues {
	return vocab.NaturalLanguageValuesNew(vocab.RefValue(vocab.NilLangRef, content))
}

func HasAttributedTo(i iri) func(*o) error {
	return func(ob *o) error {
		ob.AttributedTo = i
		return nil
	}
}

func HasContext(i iri) func(*o) error {
	return func(ob *o) error {
		ob.Context = i
		return nil
	}
}

func HasAudience(i iri) func(*o) error {
	return func(ob *o) error {
		if ob.Audience == nil {
			ob.Audience = make(ic, 0)
		}
		return ob.Audience.Append(i)
	}
}

func HasGenerator(i iri) func(*o) error {
	return func(ob *o) error {
		ob.Generator = i
		return nil
	}
}

func HasURL(i iri) func(*o) error {
	return func(ob *o) error {
		ob.URL = i
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
		if ob.Tag == nil {
			ob.Tag = make(ic, 0)
		}
		return ob.Tag.Append(t)
	}
}

func HasCC(i iri) func(*o) error {
	return func(ob *o) error {
		if ob.CC == nil {
			ob.CC = make(ic, 0)
		}
		return ob.CC.Append(i)
	}
}

func HasTo(i iri) func(*o) error {
	return func(ob *o) error {
		if ob.To == nil {
			ob.To = make(ic, 0)
		}
		return ob.To.Append(i)
	}
}

func nlv[T ~string | vocab.NaturalLanguageValues](c T) vocab.NaturalLanguageValues {
	var result vocab.NaturalLanguageValues
	switch v := any(c).(type) {
	case string:
		result = vocab.DefaultNaturalLanguage[string](v)
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

func HasPublished(s string) func(*o) error {
	p, _ := time.Parse(time.RFC3339Nano, s)
	return func(ob *o) error {
		ob.Published = p
		return nil
	}
}

func HasUpdated(s string) func(*o) error {
	u, _ := time.Parse(time.RFC3339Nano, s)
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

func HasLiked() func(*a) error {
	return func(act *a) error {
		act.Liked = vocab.Liked.IRI(act.ID)
		return nil
	}
}

func HasID(i iri) func(*o) error {
	return func(ob *o) error {
		ob.ID = i
		return nil
	}
}
func HasType(t t) func(*o) error {
	return func(ob *o) error {
		ob.Type = t
		return nil
	}
}

func Object(initFn ...InitFn) *o {
	ob := o{}
	for _, maybeFn := range initFn {
		switch fn := maybeFn.(type) {
		case func(*vocab.Object) error:
			_ = vocab.OnObject(&ob, fn)
		}
	}
	return &ob
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
