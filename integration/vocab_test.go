package integration

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

	itfn = any
)

var (
	en = vocab.DefaultNaturalLanguage[string]
)

func nl[T ~string](content T) vocab.NaturalLanguageValues {
	return vocab.NaturalLanguageValuesNew(vocab.RefValue(vocab.NilLangRef, content))
}

func hasAttributedTo(i iri) func(*o) error {
	return func(ob *o) error {
		ob.AttributedTo = i
		return nil
	}
}

func hasContext(i iri) func(*o) error {
	return func(ob *o) error {
		ob.Context = i
		return nil
	}
}

func hasAudience(i iri) func(*o) error {
	return func(ob *o) error {
		if ob.Audience == nil {
			ob.Audience = make(ic, 0)
		}
		return ob.Audience.Append(i)
	}
}

func hasGenerator(i iri) func(*o) error {
	return func(ob *o) error {
		ob.Generator = i
		return nil
	}
}

func hasURL(i iri) func(*o) error {
	return func(ob *o) error {
		ob.URL = i
		return nil
	}
}

func hasStream(i iri) func(*a) error {
	return func(ob *a) error {
		if ob.Streams == nil {
			ob.Streams = make(ic, 0)
		}
		return ob.Streams.Append(i)
	}
}

func hasPublicKey(k crypto.PublicKey) func(*a) error {
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

func hasTag(t i) func(*o) error {
	return func(ob *o) error {
		if ob.Tag == nil {
			ob.Tag = make(ic, 0)
		}
		return ob.Tag.Append(t)
	}
}

func hasCC(i iri) func(*o) error {
	return func(ob *o) error {
		if ob.CC == nil {
			ob.CC = make(ic, 0)
		}
		return ob.CC.Append(i)
	}
}

func hasTo(i iri) func(*o) error {
	return func(ob *o) error {
		if ob.To == nil {
			ob.To = make(ic, 0)
		}
		return ob.To.Append(i)
	}
}

func hasName(n string) func(*o) error {
	return func(ob *o) error {
		ob.Name = en(n)
		return nil
	}
}

func hasSummary(n string) func(*o) error {
	return func(ob *o) error {
		ob.Summary = en(n)
		return nil
	}
}

func hasSource(c string, mt string) func(*o) error {
	return func(ob *o) error {
		ob.Source.Content = nl(c)
		ob.Source.MediaType = vocab.MimeType(mt)
		return nil
	}
}

func hasContent(c string) func(*o) error {
	return func(ob *o) error {
		ob.Content = en(c)
		return nil
	}
}

func hasMediaType(m string) func(*o) error {
	return func(ob *o) error {
		ob.MediaType = vocab.MimeType(m)
		return nil
	}
}

func hasPublished(s string) func(*o) error {
	p, _ := time.Parse(time.RFC3339Nano, s)
	return func(ob *o) error {
		ob.Published = p
		return nil
	}
}

func hasUpdated(s string) func(*o) error {
	u, _ := time.Parse(time.RFC3339Nano, s)
	return func(ob *o) error {
		ob.Updated = u
		return nil
	}
}

func hasPreferredUsername(n string) func(*a) error {
	return func(act *a) error {
		act.PreferredUsername = en(n)
		return nil
	}
}

func hasSharedInbox(i iri) func(*a) error {
	return func(act *a) error {
		if act.Endpoints == nil {
			act.Endpoints = new(ep)
		}
		act.Endpoints.SharedInbox = i
		return nil
	}
}

func hasLiked() func(*a) error {
	return func(act *a) error {
		act.Liked = vocab.Liked.IRI(act.ID)
		return nil
	}
}

func hasID(i iri) func(*o) error {
	return func(ob *o) error {
		ob.ID = i
		return nil
	}
}
func hasType(t t) func(*o) error {
	return func(ob *o) error {
		ob.Type = t
		return nil
	}
}

func object(initFn ...itfn) *o {
	ob := o{}
	for _, maybeFn := range initFn {
		switch fn := maybeFn.(type) {
		case func(*vocab.Object) error:
			_ = vocab.OnObject(&ob, fn)
		}
	}
	return &ob
}

func actor(initFn ...itfn) *a {
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

func hasAuthEp(i iri) func(*a) error {
	return func(act *a) error {
		if act.Endpoints == nil {
			act.Endpoints = new(ep)
		}
		act.Endpoints.OauthAuthorizationEndpoint = i
		return nil
	}
}

func hasTokenEp(i iri) func(*a) error {
	return func(act *a) error {
		if act.Endpoints == nil {
			act.Endpoints = new(ep)
		}
		act.Endpoints.OauthTokenEndpoint = i
		return nil
	}
}
