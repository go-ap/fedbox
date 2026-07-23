package integration

import (
	"time"

	"git.sr.ht/~mariusor/lw"
	"git.sr.ht/~mariusor/storage-all"
	vocab "github.com/go-ap/activitypub"
	ap "github.com/go-ap/fedbox/integration/internal/vocab"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/fedbox/internal/env"
	"github.com/go-ap/filters"
)

var (
	defaultC2SOptions = config.Options{
		Hostname: "primary.localdomain",
		HTTPPort: 80,
		SSHPort:  422,
		Env:      env.DEV,
		LogLevel: lw.TraceLevel,
	}
	defaultS2SOptions = config.Options{
		Hostname: "secondary.localdomain",
		HTTPPort: 80,
		SSHPort:  422,
		Storage:  storage.FS,
		Env:      env.PROD,
		LogLevel: lw.DebugLevel,
	}

	c2sRootIRI = rootIRI(defaultC2SOptions)
	s2sRootIRI = rootIRI(defaultS2SOptions)

	MockDate = time.Date(2001, time.April, 1, 0, 0, 0, 00, time.UTC)
)

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

func person(actorIRI vocab.IRI, initFn ...ap.InitFn) *vocab.Actor {
	rootU, _ := actorIRI.URL()
	rootU.Path = ""
	rootIRI := vocab.IRI(rootU.String())
	initFn = append([]ap.InitFn{
		ap.HasID(actorIRI),
		ap.HasType(vocab.PersonType),
		ap.HasAttributedTo(actorIRI),
		ap.HasAudience(vocab.PublicNS),
		ap.HasGenerator(rootIRI),
		ap.HasURL(actorIRI),
		ap.HasAuthEp(vocab.CollectionPath("oauth/authorize").IRI(actorIRI)),
		ap.HasTokenEp(vocab.CollectionPath("oauth/token").IRI(actorIRI)),
		ap.HasSharedInbox(vocab.Inbox.IRI(rootIRI)),
		ap.HasProxyURL(vocab.CollectionPath("proxyUrl").IRI(rootIRI)),
	}, initFn...)
	return ap.Actor(initFn...)
}

func object(objectIRI vocab.IRI, initFn ...ap.InitFn) *vocab.Object {
	rootU, _ := objectIRI.URL()
	rootU.Path = ""
	rootIRI := vocab.IRI(rootU.String())
	initFn = append([]ap.InitFn{
		ap.HasID(objectIRI),
		ap.HasAttributedTo(rootIRI),
		ap.HasTo(vocab.PublicNS),
	}, initFn...)
	return ap.Object(initFn...)
}
