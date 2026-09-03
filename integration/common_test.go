package integration

import (
	"crypto/rand"
	"time"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/integration/internal/containers/fedbox"
	ap "github.com/go-ap/fedbox/integration/internal/vocab"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/filters"
	"golang.org/x/crypto/ed25519"
)

var (
	c2sRootIRI = rootIRI(fedbox.DefaultC2SOptions)
	s2sRootIRI = rootIRI(fedbox.DefaultS2SOptions)

	MockDate = time.Date(2001, time.April, 1, 0, 0, 0, 00, time.UTC)

	RootCreate = &vocab.Activity{
		ID:        filters.ActivitiesType.IRI(c2sRootIRI).AddPath("create-1"),
		Type:      vocab.CreateType,
		Actor:     c2sRootIRI,
		Object:    c2sRootIRI,
		Audience:  vocab.ItemCollection{vocab.PublicNS},
		Published: time.Now().Round(2 * time.Second), // NOTE(marius): to order it at the top of the collection
	}

	_, ed2559Key, _ = ed25519.GenerateKey(rand.Reader)

	tagAdmin = object(
		c2sRootIRI.AddPath("objects/0"),
		ap.HasName("#sysop"),
		ap.HasTo(vocab.PublicNS),
	)
	admin = person(
		c2sRootIRI.AddPath("actors/1"),
		ap.HasPreferredUsername("admin"),
		ap.HasAudience(vocab.PublicNS),
		ap.HasTag(tagAdmin),
		ap.HasSharedInbox(vocab.Inbox.IRI(c2sRootIRI)),
	)
	person1 = person(
		ap.HasID(c2sRootIRI.AddPath("actors/person-1")),
		ap.HasPreferredUsername("jdoe"),
		ap.HasName("John Doe"),
		ap.HasAudience(vocab.PublicNS),
		ap.HasPublished(MockDate),
		ap.HasFollowing,
		ap.HasFollowers,
		ap.HasLiked,
		ap.HasLikes,
		ap.HasShares,
		ap.HasReplies,
		ap.HasSharedInbox(vocab.Inbox.IRI(c2sRootIRI)),
	)
	person3 = person(
		c2sRootIRI.AddPath("actors/person-3"),
		ap.HasPreferredUsername("alice"),
		ap.HasContent("lorem ipsum dolor sic amet"),
		ap.HasAudience(vocab.PublicNS),
		ap.HasPublished(MockDate),
		ap.HasFollowing,
		ap.HasFollowers,
		ap.HasShares,
	)
)

func errFedBOXNotFound(iri vocab.IRI) error {
	u, _ := iri.URL()
	return errors.NotFoundf("%s not found", u.Path)
}

func rootIRI(conf config.Options) vocab.IRI {
	return vocab.IRI("http://" + conf.Hostname)
}

func root(rootIRI vocab.IRI, initFn ...ap.InitFn) *vocab.Actor {
	initFn = append([]ap.InitFn{
		ap.HasID(rootIRI),
		ap.HasType(vocab.ServiceType),
		ap.HasPreferredUsername("FedBOX"),
		ap.HasAttributedTo("https://github.com/mariusor"),
		ap.HasAudience(vocab.PublicNS),
		ap.HasContext("https://github.com/go-ap/fedbox"),
		ap.HasSummary("Generic ActivityPub service"),
		ap.HasURL(rootIRI),
		ap.HasStream(filters.ActorsType.IRI(rootIRI)),
		ap.HasStream(filters.ActivitiesType.IRI(rootIRI)),
		ap.HasStream(filters.ObjectsType.IRI(rootIRI)),
		ap.HasAuthEp(vocab.CollectionPath("/oauth/authorize").IRI(rootIRI)),
		ap.HasTokenEp(vocab.CollectionPath("/oauth/token").IRI(rootIRI)),
		ap.HasProxyURL(vocab.CollectionPath("proxyUrl").IRI(rootIRI)),
	}, initFn...)

	return ap.Actor(initFn...)
}

func create(initFn ...ap.InitFn) *vocab.Activity {
	initFn = append([]ap.InitFn{
		ap.HasType(vocab.CreateType),
	}, initFn...)
	return ap.Activity(initFn...)
}

func update(initFn ...ap.InitFn) *vocab.Activity {
	initFn = append([]ap.InitFn{
		ap.HasType(vocab.UpdateType),
	}, initFn...)
	return ap.Activity(initFn...)
}

func del(initFn ...ap.InitFn) *vocab.Activity {
	initFn = append([]ap.InitFn{
		ap.HasType(vocab.DeleteType),
	}, initFn...)
	return ap.Activity(initFn...)
}

func like(initFn ...ap.InitFn) *vocab.Activity {
	initFn = append([]ap.InitFn{
		ap.HasType(vocab.LikeType),
	}, initFn...)
	return ap.Activity(initFn...)
}

func announce(initFn ...ap.InitFn) *vocab.Activity {
	initFn = append([]ap.InitFn{
		ap.HasType(vocab.AnnounceType),
	}, initFn...)
	return ap.Activity(initFn...)
}

func follow(initFn ...ap.InitFn) *vocab.Activity {
	initFn = append([]ap.InitFn{
		ap.HasType(vocab.FollowType),
	}, initFn...)
	return ap.Activity(initFn...)
}

func accept(initFn ...ap.InitFn) *vocab.Activity {
	initFn = append([]ap.InitFn{
		ap.HasType(vocab.AcceptType),
	}, initFn...)
	return ap.Activity(initFn...)
}

func reject(initFn ...ap.InitFn) *vocab.Activity {
	initFn = append([]ap.InitFn{
		ap.HasType(vocab.RejectType),
	}, initFn...)
	return ap.Activity(initFn...)
}

func block(initFn ...ap.InitFn) *vocab.Activity {
	initFn = append([]ap.InitFn{
		ap.HasType(vocab.BlockType),
	}, initFn...)
	return ap.Activity(initFn...)
}

func ignore(initFn ...ap.InitFn) *vocab.Activity {
	initFn = append([]ap.InitFn{
		ap.HasType(vocab.IgnoreType),
	}, initFn...)
	return ap.Activity(initFn...)
}

func undo(initFn ...ap.InitFn) *vocab.Activity {
	initFn = append([]ap.InitFn{
		ap.HasType(vocab.UndoType),
	}, initFn...)
	return ap.Activity(initFn...)
}

func question(initFn ...ap.InitFn) *vocab.Question {
	initFn = append([]ap.InitFn{
		ap.HasType(vocab.QuestionType),
		ap.HasAudience(vocab.PublicNS),
	}, initFn...)
	return ap.Question(initFn...)
}

func add(initFn ...ap.InitFn) *vocab.Add {
	initFn = append([]ap.InitFn{
		ap.HasType(vocab.AddType),
		ap.HasAudience(vocab.PublicNS),
	}, initFn...)
	return ap.Activity(initFn...)
}

func remove(initFn ...ap.InitFn) *vocab.Activity {
	initFn = append([]ap.InitFn{
		ap.HasType(vocab.RemoveType),
		ap.HasAudience(vocab.PublicNS),
	}, initFn...)
	return ap.Activity(initFn...)
}

func move(initFn ...ap.InitFn) *vocab.Activity {
	initFn = append([]ap.InitFn{
		ap.HasType(vocab.MoveType),
		ap.HasAudience(vocab.PublicNS),
	}, initFn...)
	return ap.Activity(initFn...)
}

func baseIRI(iri vocab.IRI) vocab.IRI {
	ub, err := iri.GetLink().URL()
	if err != nil {
		return iri
	}
	ub.Path = ""
	ub.RawQuery = ""
	ub.Fragment = ""
	return vocab.IRI(ub.String())
}

func person(initFn ...ap.InitFn) *vocab.Actor {
	var actorIRI vocab.IRI
	for _, maybeFn := range initFn {
		if iri, ok := maybeFn.(vocab.IRI); ok {
			actorIRI = iri
			break
		}
	}
	initFn = append([]ap.InitFn{
		ap.HasID(actorIRI),
		ap.HasType(vocab.PersonType),
		ap.HasAttributedTo(actorIRI),
		ap.HasAudience(vocab.PublicNS),
		ap.HasURL(actorIRI),
		ap.HasAuthEp(vocab.CollectionPath("oauth/authorize").IRI(actorIRI)),
		ap.HasTokenEp(vocab.CollectionPath("oauth/token").IRI(actorIRI)),
	}, initFn...)
	if serviceIRI := baseIRI(actorIRI); serviceIRI != "" {
		initFn = append(initFn,
			ap.HasGenerator(serviceIRI),
			ap.HasProxyURL(vocab.CollectionPath("proxyUrl").IRI(serviceIRI)),
		)
	}
	return ap.Actor(initFn...)
}

func object(initFn ...ap.InitFn) *vocab.Object {
	var objectIRI vocab.IRI
	for _, maybeFn := range initFn {
		if iri, ok := maybeFn.(vocab.IRI); ok {
			objectIRI = iri
			break
		}
	}

	if objectIRI != "" {
		initFn = append(initFn, ap.HasID(objectIRI))
	}
	return ap.Object(initFn...)
}

func orderedCollection(initFn ...ap.InitFn) *vocab.OrderedCollection {
	var objectIRI vocab.IRI
	for _, maybeFn := range initFn {
		if iri, ok := maybeFn.(vocab.IRI); ok {
			objectIRI = iri
			break
		}
	}
	if objectIRI != "" {
		initFn = append(initFn, ap.HasID(objectIRI), ap.HasType(vocab.OrderedCollectionType))
	}

	return ap.OrderedCollection(initFn...)
}

func collection(initFn ...ap.InitFn) *vocab.Collection {
	var objectIRI vocab.IRI
	for _, maybeFn := range initFn {
		if iri, ok := maybeFn.(vocab.IRI); ok {
			objectIRI = iri
			break
		}
	}
	if objectIRI != "" {
		initFn = append(initFn, ap.HasID(objectIRI), ap.HasType(vocab.CollectionType))
	}

	return ap.Collection(initFn...)
}

func filterIRI(iri vocab.IRI, ff ...filters.Check) vocab.IRI {
	if filters.MaxCountCheck(ff...) == nil {
		// NOTE(marius): the FedBOX server appends a maxItems filter of 100 if it's missing.
		ff = append(ff, filters.WithMaxCount(filters.MaxItems))
	}
	return filters.IRIf(iri, ff...)
}
