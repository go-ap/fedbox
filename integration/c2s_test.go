package integration

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"strings"
	"testing"
	"time"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/client"
	"github.com/go-ap/client/c2s"
	"github.com/go-ap/client/s2s"
	"github.com/go-ap/errors"
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

	MockDate = time.Date(2001, time.April, 1, 0, 0, 23, 00, time.UTC)
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

	fedbox := service(c2sRootIRI, ap.HasPublicKey(publicKey))

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
			Req:  tests.URL(c2sRootIRI),
			Res:  tests.Response(tests.HasCode(http.StatusOK), tests.HasContentType(contentTypes...), tests.HasExactItem(fedbox)),
		},
		{
			Name: "actors/1",
			Req:  tests.URL(admin1.ID),
			Res:  tests.Response(tests.HasCode(http.StatusOK), tests.HasContentType(contentTypes...), tests.HasExactItem(admin1)),
		},
		{
			Name: "objects/0",
			Req:  tests.URL(tag0.ID),
			Res:  tests.Response(tests.HasCode(http.StatusOK), tests.HasContentType(contentTypes...), tests.HasExactItem(tag0)),
		},
		{
			Name: "objects/1",
			Req:  tests.URL(object1.ID),
			Res:  tests.Response(tests.HasCode(http.StatusOK), tests.HasContentType(contentTypes...), tests.HasExactItem(object1)),
		},
		{
			Name: "actors/2",
			Req:  tests.URL(actor2.ID),
			Res:  tests.Response(tests.HasCode(http.StatusOK), tests.HasContentType(contentTypes...), tests.HasExactItem(actor2)),
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

	c2sFedBOX := c.FedBOXNew(
		c.WithImageName(fedBOXImageName),
		c.WithConfig(c.ConfigFromBuildInfo(defaultC2SOptions)),
		c.WithArgs([]string{"--bootstrap"}),
		c.WithRootIRI(c2sRootIRI),
		c.WithKey(prvKey), c.WithPw(rand.Text()[:8]),
		c.WithItems(tagAdmin, admin),
		c.WithTestLogger(t, Verbose),
	)

	contentTypes := []string{client.ContentTypeJsonLD, client.ContentTypeJsonActivity}
	toRun := []tests.RunnableTest{
		tests.HTTPTest{
			Name: "tag admin",
			Req:  tests.URL(string(tagAdmin.ID)),
			Res:  tests.Response(tests.HasCode(http.StatusOK), tests.HasContentType(contentTypes...), tests.HasExactItem(tagAdmin)),
		},
		tests.HTTPTest{
			Name: "admin",
			Req:  tests.URL(admin.ID),
			Res:  tests.Response(tests.HasCode(http.StatusOK), tests.HasContentType(contentTypes...), tests.HasExactItem(admin)),
		},
		tests.HTTPTest{
			Name: "invalid body",
			Req: tests.URL(admin.Inbox.GetLink()).
				Post().
				Header("Content-Type", client.ContentTypeJsonLD).
				BodyBytes(nil),
			Res: tests.Response(
				tests.HasCode(http.StatusBadRequest),
				tests.HasContentType(client.ContentTypeJson),
				tests.HasErrors(errors.BadRequestf("unable to unmarshal JSON request")),
			),
		},
		tests.HTTPTest{
			Name: "non authorized",
			Req: tests.URL(admin.Inbox.GetLink()).
				Post().
				Header("Content-Type", client.ContentTypeJsonLD).
				BodyBytes([]byte(`{"type":"Flag"}`)),
			Res: tests.Response(
				tests.HasCode(http.StatusUnauthorized),
				tests.HasContentType(client.ContentTypeJson),
				tests.HasErrors(errors.Unauthorizedf("unable to read request body")),
			),
		},
		tests.HTTPTest{
			Name: "collection not found",
			Req: tests.URL(c2sRootIRI.AddPath("test")).
				Post().
				Header("Content-Type", client.ContentTypeJsonLD).
				BodyBytes([]byte(`{"type":"Flag"}`)),
			Res: tests.Response(
				tests.HasCode(http.StatusNotFound),
				tests.HasContentType(client.ContentTypeJson),
				tests.HasErrors(errors.NotFoundf("invalid collection")),
			),
		},
		tests.HTTPTest{
			Name: "to inbox",
			Req: tests.URL(admin.Inbox.GetLink()).
				Post().
				ContentType(client.ContentTypeJsonLD).
				Header("Date", MockDate.Format(http.TimeFormat)).
				Signer(draftSig.SignDraft).
				BodyBytes([]byte(`{"type":"Flag"}`)),
			Res: tests.Response(
				tests.HasCode(http.StatusBadRequest),
				tests.HasContentType(client.ContentTypeJson),
				tests.HasErrors(
					errors.NewBadRequest(errors.BadRequestf("Activity is not valid: invalid activity id"), "Unable to save activity Flag to http://fedbox/actors/1/inbox"),
					errors.BadRequestf("Activity is not valid: invalid activity id"),
				),
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
			IO: tests.WithTests(func(t *testing.T, i []byte) []byte {
				i = bytes.TrimSpace(i)
				auth, found := bytes.CutPrefix(i, []byte("Authorization: "))
				if !found {
					t.Fatalf("Unable to get Authorization value from CLI output: %s", i)
				}
				authPieces := strings.Split(string(auth), " ")
				if len(authPieces) < 2 {
					t.Fatalf("Authorization value is not recognized: %+v", authPieces)
				}
				token.TokenType = authPieces[0]
				token.AccessToken = authPieces[1]
				if token.AccessToken == "" || token.TokenType == "" {
					t.Fatalf("Unable to build Authorization token")
				}
				return nil
			}, endOK),
		},
		tests.HTTPTest{
			Name: "to outbox",
			Req: tests.URL(admin.Outbox.GetLink()).
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
			Res: tests.Response(
				tests.HasCode(http.StatusCreated),
				tests.HasContentType(contentTypes...),
				tests.HasLocation(admin.ID),
				tests.HasItemProperties(&vocab.Activity{Type: vocab.FlagType, Actor: admin, Object: admin, AttributedTo: admin, Published: MockDate}),
			),
		},
	}

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
		t.Run(test.Label(), test.Fn(ctx, cont))
	}
}
