package integration

import (
	"crypto"
	"crypto/rand"
	"time"

	vocab "github.com/go-ap/activitypub"
	ap "github.com/go-ap/activitypub/build"
	"github.com/go-ap/errors"
	ap2 "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/fedbox/integration/internal/containers/fedbox"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/filters"
	"github.com/go-ap/storage-conformance-suite/gen"
	"golang.org/x/crypto/ed25519"
)

var (
	c2sRootIRI = rootIRI(fedbox.DefaultC2SOptions)
	s2sRootIRI = rootIRI(fedbox.DefaultS2SOptions)

	MockDate = time.Date(2001, time.April, 1, 0, 0, 0, 00, time.UTC)

	RootCreate = ap.Create(
		ap.WithID(filters.ActivitiesType.IRI(c2sRootIRI).AddPath("create-1")),
		ap.WithType(vocab.CreateType),
		ap.WithActor(c2sRootIRI),
		ap.WithObject(c2sRootIRI),
		ap.WithAudience(vocab.PublicNS),
		ap.WithPublished(time.Now().Round(2*time.Second)), // NOTE(marius): to order it at the top of the collection
	)

	_, ed2559Key, _ = ed25519.GenerateKey(rand.Reader)

	tagAdmin = ap.Object(
		ap.WithID(c2sRootIRI.AddPath("objects/0")),
		ap.WithName("#sysop"),
		ap.WithAttributedTo(c2sRootIRI),
		ap.WithTo(vocab.PublicNS),
		ap.WithPublished(MockDate),
		ap.WithUpdated(MockDate),
		ap.WithCC(c2sRootIRI),
	)

	admin = ap.Person(
		ap.WithID(c2sRootIRI.AddPath("actors/1")),
		ap.WithPreferredUsername("admin"),
		ap.WithAttributedTo(c2sRootIRI.AddPath("actors/1")),
		ap.WithAudience(vocab.PublicNS),
		ap.WithTag(tagAdmin.ID),
		ap.WithInbox(),
		ap.WithOutbox(),
		ap.WithSharedInbox(vocab.Inbox.IRI(c2sRootIRI)),
	)

	person1 = ap.Person(
		ap.WithID(c2sRootIRI.AddPath("actors/person-1")),
		ap.WithPreferredUsername("jdoe"),
		ap.WithName("John Doe"),
		ap.WithAudience(vocab.PublicNS),
		ap.WithPublished(MockDate),
		ap.WithInbox(),
		ap.WithOutbox(),
		ap.WithFollowing(),
		ap.WithFollowers(),
		ap.WithLiked(),
		ap.WithLikes(),
		ap.WithShares(),
		ap.WithReplies(),
		ap.WithSharedInbox(vocab.Inbox.IRI(c2sRootIRI)),
	)

	person3 = ap.Person(
		ap.WithID(c2sRootIRI.AddPath("actors/person-3")),
		ap.WithPreferredUsername("alice"),
		ap.WithContent("lorem ipsum dolor sic amet"),
		ap.WithAudience(vocab.PublicNS),
		ap.WithPublished(MockDate),
		ap.WithInbox(),
		ap.WithOutbox(),
		ap.WithFollowing(),
		ap.WithFollowers(),
		ap.WithShares(),
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
		ap.WithID(rootIRI),
		ap.WithPreferredUsername("FedBOX"),
		ap.WithAttributedTo(vocab.IRI("https://github.com/mariusor")),
		ap.WithAudience(vocab.PublicNS),
		ap.WithContext(vocab.IRI("https://github.com/go-ap/fedbox")),
		ap.WithSummary("Generic ActivityPub service"),
		ap.WithURL(rootIRI),
		ap.WithInbox(vocab.Inbox.IRI(rootIRI)),
		ap.WithOutbox(vocab.Outbox.IRI(rootIRI)),
		ap.WithStreams(
			filters.ActorsType.IRI(rootIRI),
			filters.ActivitiesType.IRI(rootIRI),
			filters.ObjectsType.IRI(rootIRI),
		),
		ap.WithAuthEndpoint(vocab.CollectionPath("/oauth/authorize").IRI(rootIRI)),
		ap.WithTokenEndpoint(vocab.CollectionPath("/oauth/token").IRI(rootIRI)),
		ap.WithProxyURL(vocab.CollectionPath("proxyUrl").IRI(rootIRI)),
	}, initFn...)

	return ap.Service(initFn...)
}

func filterIRI(iri vocab.IRI, ff ...filters.Check) vocab.IRI {
	if filters.MaxCountCheck(ff...) == nil {
		// NOTE(marius): the FedBOX server appends a maxItems filter of 100 if it's missing.
		ff = append(ff, filters.WithMaxCount(filters.MaxItems))
	}
	return filters.IRIf(iri, ff...)
}

func plausibleRandomObjects(pubKey crypto.PublicKey, cnt int) vocab.ItemCollection {
	service := root(c2sRootIRI, ap.WithPublicKey(ap.PublicKey("", pubKey)))

	gen.DefaultHost = c2sRootIRI
	gen.SetItemID = func(it vocab.Item) {
		ap2.GenerateID(it, c2sRootIRI, service)
	}
	gen.RootID = service.ID
	gen.Root = service

	randomObjects := make(vocab.ItemCollection, 0, cnt)
	for _, ob := range gen.PlausibleStorage(gen.Root, cnt) {
		if it, ok := ob.(vocab.Item); ok {
			_ = randomObjects.Append(it)
		}
	}

	return randomObjects
}
