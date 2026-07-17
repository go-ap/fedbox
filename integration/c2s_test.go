package integration

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"testing"

	vocab "github.com/go-ap/activitypub"
	c "github.com/go-ap/fedbox/integration/internal/containers"
	"github.com/go-ap/fedbox/integration/internal/tests"
	ap "github.com/go-ap/fedbox/integration/internal/vocab"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/filters"
	"golang.org/x/crypto/ed25519"
)

var (
	defaultC2SOptions = config.Options{
		Hostname: "fedbox",
		HTTPPort: 4000,
		SSHPort:  4044,
	}

	defaultS2SOptions = config.Options{
		Hostname: "secondary",
		HTTPPort: 4011,
		SSHPort:  4055,
	}
)

func rootIRI(conf config.Options) vocab.IRI {
	return vocab.IRI("http://" + conf.Hostname)
}

var c2sRootIRI = rootIRI(defaultC2SOptions)
var s2sRootIRI = rootIRI(defaultS2SOptions)

func service(rootIRI vocab.IRI, initFn ...ap.InitFn) *vocab.Actor {
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

var admin1 = person("http://fedbox/actors/1",
	ap.HasPreferredUsername("admin"),
	ap.HasTag(tag0),
)

var actor2 = person("http://fedbox/actors/2",
	ap.HasContent("Generated actor"),
	ap.HasSummary("Generated actor"),
	ap.HasLiked(),
	ap.HasPreferredUsername("johndoe"),
	ap.HasPublished("2019-08-11T13:14:47.000000000+02:00"),
	ap.HasUpdated("2019-08-11T13:14:47.000000000+02:00"),
	ap.HasName("Johnathan Doe"),
)

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

var tag0 = object(vocab.CollectionPath("objects/0").IRI(c2sRootIRI), ap.HasName("#sysop"))

var object1 = object(vocab.CollectionPath("objects/1").IRI(c2sRootIRI),
	ap.HasType(vocab.NoteType),
	ap.HasContent("<p>Hello</p><code>FedBOX</code>!</p>\n"),
	ap.HasMediaType("text/html"),
	ap.HasPublished("2019-09-27T14:26:43.000000000Z"),
	ap.HasUpdated("2019-09-27T14:26:43.000000000Z"),
	ap.HasAttributedTo(admin1.ID),
	ap.HasSource("Hello `FedBOX`!", "text/markdown"),
	ap.HasTo("https://www.w3.org/ns/activitystreams#Public"),
)

func Test_Fetch(t *testing.T) {
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	var fedbox = service("http://fedbox", ap.HasPublicKey(publicKey))

	toRun := []tests.HTTPTest{
		{
			Name: "service",
			Req:  tests.Request(tests.WithURL(c2sRootIRI)),
			Res:  tests.Response(tests.HasCode(http.StatusOK), tests.HasItem(fedbox)),
		},
		{
			Name: "actors/1",
			Req:  tests.Request(tests.WithURL(admin1.ID)),
			Res:  tests.Response(tests.HasCode(http.StatusOK), tests.HasItem(admin1)),
		},
		{
			Name: "objects/0",
			Req:  tests.Request(tests.WithURL(tag0.ID)),
			Res:  tests.Response(tests.HasCode(http.StatusOK), tests.HasItem(tag0)),
		},
		{
			Name: "objects/1",
			Req:  tests.Request(tests.WithURL(object1.ID)),
			Res:  tests.Response(tests.HasCode(http.StatusOK), tests.HasItem(object1)),
		},
		{
			Name: "actors/2",
			Req:  tests.Request(tests.WithURL(actor2.ID)),
			Res:  tests.Response(tests.HasCode(http.StatusOK), tests.HasItem(actor2)),
		},
	}

	c2sFedBOX := c.FedBOXNew(
		c.WithConfig(c.ConfigFromBuildInfo(defaultC2SOptions)),
		c.WithArgs([]string{"--bootstrap"}),
		c.WithTestLogger(t, Verbose),
		c.WithImageName(fedBOXImageName),
		c.WithRootIRI(c2sRootIRI),
		c.WithKey(privateKey),
		c.WithPw(rand.Text()[:8]),
		c.WithItems(tag0, object1, admin1, actor2),
	)

	images := c.Suite{c2sFedBOX}

	ctx := context.Background()
	cont, err := c.Init(ctx, t, images...)
	if err != nil {
		t.Fatalf("unable to initialize containers: %s", err)
	}

	t.Cleanup(func() {
		cont.Cleanup(t)
	})

	for _, test := range toRun {
		t.Run(test.Name, test.Fn(ctx, cont))
	}
}

func Test_MoreComplicated(t *testing.T) {
	t.Skip()
	_, c2sPrvKey, _ := ed25519.GenerateKey(rand.Reader)

	c2sTagAdmin := object(c2sRootIRI.AddPath("objects/0"), ap.HasName("#sysop"))
	c2sAdmin := person(c2sRootIRI.AddPath("actors/1"), ap.HasPreferredUsername("admin"), ap.HasTag(c2sTagAdmin))

	s2sPrvKey, _ := rsa.GenerateKey(rand.Reader, 1024)

	s2sTagAdmin := object(s2sRootIRI.AddPath("objects/0"), ap.HasName("#sysop"))
	s2sAdmin := person(vocab.CollectionPath("actors/1").IRI(s2sRootIRI), ap.HasPreferredUsername("admin"), ap.HasTag(s2sTagAdmin))

	toRun := []tests.RunnableTest{
		tests.HTTPTest{
			Name: "c2s tag admin",
			Req:  tests.Request(tests.WithURL(c2sTagAdmin.ID)),
			Res:  tests.Response(tests.HasCode(http.StatusOK), tests.HasItem(c2sTagAdmin)),
		},
		tests.HTTPTest{
			Name: "s2s tag admin",
			Req:  tests.Request(tests.WithURL(s2sTagAdmin.ID)),
			Res:  tests.Response(tests.HasCode(http.StatusOK), tests.HasItem(s2sTagAdmin)),
		},
		tests.HTTPTest{
			Name: "c2s admin",
			Req:  tests.Request(tests.WithURL(c2sAdmin.ID)),
			Res:  tests.Response(tests.HasCode(http.StatusOK), tests.HasItem(c2sAdmin)),
		},
		tests.HTTPTest{
			Name: "s2s admin",
			Req:  tests.Request(tests.WithURL(s2sAdmin.ID)),
			Res:  tests.Response(tests.HasCode(http.StatusOK), tests.HasItem(s2sAdmin)),
		},
	}

	c2sFedBOX := c.FedBOXNew(
		c.WithConfig(c.ConfigFromBuildInfo(defaultC2SOptions)),
		c.WithImageName(fedBOXImageName),
		c.WithKey(c2sPrvKey),
		c.WithRootIRI(c2sRootIRI),
		c.WithPw(rand.Text()[:8]),
		c.WithItems(c2sTagAdmin, c2sAdmin),
	)
	s2sFedBOX := c.FedBOXNew(
		c.WithConfig(c.ConfigFromBuildInfo(defaultS2SOptions)),
		c.WithImageName(fedBOXImageName),
		c.WithKey(s2sPrvKey),
		c.WithRootIRI(s2sRootIRI),
		c.WithPw(rand.Text()[:8]),
		c.WithItems(s2sTagAdmin, s2sAdmin),
	)

	images := c.Suite{c2sFedBOX, s2sFedBOX}

	ctx := context.Background()
	cont, err := c.Init(ctx, t, images...)
	if err != nil {
		t.Fatalf("unable to initialize containers: %s", err)
	}

	t.Cleanup(func() {
		cont.Cleanup(t)
	})

	for _, test := range toRun {
		t.Run(test.Label(), test.Fn(ctx, cont))
	}
}
