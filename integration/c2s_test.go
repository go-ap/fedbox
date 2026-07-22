package integration

import (
	"context"
	"crypto/rand"
	"net/http"
	"testing"
	"time"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/client"
	"github.com/go-ap/client/c2s"
	"github.com/go-ap/client/s2s"
	"github.com/go-ap/errors"
	c "github.com/go-ap/fedbox/integration/internal/containers"
	"github.com/go-ap/fedbox/integration/internal/containers/fedbox"
	"github.com/go-ap/fedbox/integration/internal/tests"
	ap "github.com/go-ap/fedbox/integration/internal/vocab"
	"github.com/go-ap/fedbox/internal/config"
	"github.com/go-ap/filters"
	"golang.org/x/crypto/ed25519"
)

var (
	defaultC2SOptions = config.Options{
		Hostname: "first1",
		HTTPPort: 4000,
		SSHPort:  4044,
	}

	MockDate = time.Date(2001, time.April, 1, 0, 0, 23, 00, time.UTC)
)

func rootIRI(conf config.Options) vocab.IRI {
	return vocab.IRI("http://" + conf.Hostname)
}

var c2sRootIRI = rootIRI(defaultC2SOptions)
var s2sRootIRI = rootIRI(defaultS2SOptions)

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

func Test_Fetch(t *testing.T) {
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	service := root(c2sRootIRI, ap.HasPublicKey(publicKey))

	tag0 := object(c2sRootIRI.AddPath("objects/0"), ap.HasName("#sysop"))

	admin1 := person(c2sRootIRI.AddPath("actors/1"),
		ap.HasPreferredUsername("admin"),
		ap.HasTag(tag0),
	)

	actor2 := person(c2sRootIRI.AddPath("actors/2"),
		ap.HasContent("Generated actor"),
		ap.HasSummary("Generated actor"),
		ap.HasLiked(),
		ap.HasPreferredUsername("johndoe"),
		ap.HasPublished("2019-08-11T13:14:47.000000000+02:00"),
		ap.HasUpdated("2019-08-11T13:14:47.000000000+02:00"),
		ap.HasName("Johnathan Doe"),
	)

	object1 := object(c2sRootIRI.AddPath("objects/1"),
		ap.HasType(vocab.NoteType),
		ap.HasContent("<p>Hello</p><code>FedBOX</code>!</p>\n"),
		ap.HasMediaType("text/html"),
		ap.HasPublished("2019-09-27T14:26:43.000000000Z"),
		ap.HasUpdated("2019-09-27T14:26:43.000000000Z"),
		ap.HasAttributedTo(admin1.ID),
		ap.HasSource("Hello `FedBOX`!", "text/markdown"),
		ap.HasTo("https://www.w3.org/ns/activitystreams#Public"),
	)

	contentTypes := []string{client.ContentTypeJsonLD, client.ContentTypeJsonActivity}
	toRun := []tests.HTTPTest{
		{
			Name: "service",
			Req:  tests.Request().IRI(c2sRootIRI),
			Res:  tests.Response().HasCode(http.StatusOK).HasContentType(contentTypes...).HasExactItem(service),
		},
		{
			Name: "actors/1",
			Req:  tests.Request().IRI(admin1.ID),
			Res:  tests.Response().HasCode(http.StatusOK).HasContentType(contentTypes...).HasExactItem(admin1),
		},
		{
			Name: "objects/0",
			Req:  tests.Request().IRI(tag0.ID),
			Res:  tests.Response().HasCode(http.StatusOK).HasContentType(contentTypes...).HasExactItem(tag0),
		},
		{
			Name: "objects/1",
			Req:  tests.Request().IRI(object1.ID),
			Res:  tests.Response().HasCode(http.StatusOK).HasContentType(contentTypes...).HasExactItem(object1),
		},
		{
			Name: "actors/2",
			Req:  tests.Request().IRI(actor2.ID),
			Res:  tests.Response().HasCode(http.StatusOK).HasExactItem(actor2),
		},
	}

	images := c.Suite(fedbox.New(
		fedbox.WithConfig(fedbox.ConfigFromBuildInfo(defaultC2SOptions)),
		fedbox.WithArgs([]string{"--bootstrap"}),
		fedbox.WithTestLogger(t, Verbose),
		fedbox.WithImageName(fedBOXImageName),
		fedbox.WithRootIRI(c2sRootIRI),
		fedbox.WithKey(privateKey),
		fedbox.WithPw(rand.Text()[:8]),
		fedbox.WithItems(tag0, object1, admin1, actor2),
	))

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

func Test_C2S_Requests(t *testing.T) {
	_, prvKey, _ := ed25519.GenerateKey(rand.Reader)
	_, admKey, _ := ed25519.GenerateKey(rand.Reader)

	tagAdmin := object(c2sRootIRI.AddPath("objects/0"), ap.HasName("#sysop"))
	admin := person(
		c2sRootIRI.AddPath("actors/1"),
		ap.HasPreferredUsername("admin"),
		ap.HasTag(tagAdmin),
		ap.HasPublicKey(admKey.Public()),
	)

	draftSig := s2s.New(s2s.WithActor(admin, admKey))
	token := new(c2s.BearerSigner)

	contentTypes := []string{client.ContentTypeJsonLD, client.ContentTypeJsonActivity}
	toRun := []tests.RunnableTest{
		tests.HTTPTest{
			Name: "tag admin",
			Req:  tests.Request().IRI(tagAdmin.ID),
			Res:  tests.Response().HasCode(http.StatusOK).HasContentType(contentTypes...).HasExactItem(tagAdmin),
		},
		tests.HTTPTest{
			Name: "admin",
			Req:  tests.Request().IRI(admin.ID),
			Res:  tests.Response().HasCode(http.StatusOK).HasContentType(contentTypes...).HasExactItem(admin),
		},
		tests.HTTPTest{
			Name: "invalid body",
			Req: tests.Request().IRI(admin.Inbox.GetLink()).
				Post().
				Header("Content-Type", client.ContentTypeJsonLD).
				BodyBytes(nil),
			Res: tests.Response().
				HasCode(http.StatusBadRequest).
				HasContentType(client.ContentTypeJson).
				HasErrors(errors.BadRequestf("unable to unmarshal JSON request")),
		},
		tests.HTTPTest{
			Name: "non authorized",
			Req: tests.Request().IRI(admin.Inbox.GetLink()).
				Post().
				Header("Content-Type", client.ContentTypeJsonLD).
				BodyBytes([]byte(`{"type":"Flag"}`)),
			Res: tests.Response().
				HasCode(http.StatusUnauthorized).
				HasContentType(client.ContentTypeJson).
				HasErrors(errors.Unauthorizedf("unable to read request body")),
		},
		tests.HTTPTest{
			Name: "collection not found",
			Req: tests.Request().IRI(c2sRootIRI.AddPath("test")).
				Post().
				Header("Content-Type", client.ContentTypeJsonLD).
				BodyBytes([]byte(`{"type":"Flag"}`)),
			Res: tests.Response().
				HasCode(http.StatusNotFound).
				HasContentType(client.ContentTypeJson).
				HasErrors(errors.NotFoundf("invalid collection")),
		},
		tests.HTTPTest{
			Name: "to inbox",
			Req: tests.Request().IRI(admin.Inbox.GetLink()).
				Post().
				ContentType(client.ContentTypeJsonLD).
				Header("Date", MockDate.Format(http.TimeFormat)).
				Signer(draftSig.SignDraft).
				BodyBytes([]byte(`{"type":"Flag"}`)),
			Res: tests.Response().
				HasCode(http.StatusBadRequest).
				HasContentType(client.ContentTypeJson).
				HasErrors(
					errors.NewBadRequest(errors.BadRequestf("Activity is not valid: invalid activity id"), "Unable to save activity Flag to http://fedbox/actors/1/inbox"),
					errors.BadRequestf("Activity is not valid: invalid activity id"),
				),
		},
		tests.CommandTest{
			Name: "gen OAuth2 bearer",
			Host: string(c2sRootIRI),
			Cmd: c.SSHCmd{
				Cmd:  []string{"oauth", "token", "add", string(admin.ID)},
				User: c2sRootIRI.String(),
				Key:  prvKey,
			},
			IO: tests.WithTests(extractToken(token), anyOutput),
		},
		tests.HTTPTest{
			Name: "to outbox",
			Req: tests.Request().IRI(admin.Outbox.GetLink()).
				Post().
				ContentType(client.ContentTypeJsonLD).
				Signer(func(r *http.Request) error {
					auth := token.TokenType + " " + token.AccessToken
					if len(auth) <= 1 {
						t.Fatalf("Invalid authorization token: %v", token)
					}
					r.Header.Set("Authorization", auth)
					return nil
				}).
				BodyBytes([]byte(`{"type":"Flag","actor":"http://fedbox/actors/1","object":"http://fedbox/actors/1","published":"2001-04-01T00:00:23Z"}`)),
			Res: tests.Response().
				HasCode(http.StatusCreated).
				HasContentType(contentTypes...).
				HasLocation(admin.ID).
				HasItemProperties(&vocab.Activity{Type: vocab.FlagType, Actor: admin, Object: admin, AttributedTo: admin, Published: MockDate}),
		},
	}

	images := c.Suite(fedbox.New(
		fedbox.WithImageName(fedBOXImageName),
		fedbox.WithConfig(fedbox.ConfigFromBuildInfo(defaultC2SOptions)),
		fedbox.WithArgs([]string{"--bootstrap"}),
		fedbox.WithRootIRI(c2sRootIRI),
		fedbox.WithKey(prvKey), fedbox.WithPw(rand.Text()[:8]),
		fedbox.WithItems(tagAdmin, admin),
		fedbox.WithTestLogger(t, Verbose),
	))

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
