//go:build c2s || Fetch

package integration

import (
	"crypto/rand"
	"net/http"
	"testing"
	"time"

	vocab "github.com/go-ap/activitypub"
	ap "github.com/go-ap/activitypub/build"
	"github.com/go-ap/client"
	"github.com/go-ap/fedbox/integration/internal/containers/fedbox"
	"github.com/go-ap/fedbox/integration/internal/tests"
	"github.com/go-ap/filters"
	"golang.org/x/crypto/ed25519"
)

func Test_Fetch(t *testing.T) {
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	service := root(c2sRootIRI, ap.WithPublicKey(ap.PublicKey(c2sRootIRI+"#main", publicKey)), ap.WithTo(vocab.PublicNS), ap.WithCC(c2sRootIRI))

	tag0 := ap.Object(
		ap.WithID(c2sRootIRI.AddPath("objects/0")),
		ap.WithName("#sysop"),
		ap.WithTo(vocab.PublicNS),
	)

	admin1 := ap.Person(
		ap.WithID(c2sRootIRI.AddPath("actors/1")),
		ap.WithPreferredUsername("admin"),
		ap.WithInbox(),
		ap.WithOutbox(),
		ap.WithTag(tag0),
	)

	actor2 := ap.Person(
		ap.WithID(c2sRootIRI.AddPath("actors/2")),
		ap.WithContent("Generated actor"),
		ap.WithSummary("Generated actor"),
		ap.WithAttributedTo(c2sRootIRI),
		ap.WithInbox(),
		ap.WithOutbox(),
		ap.WithLiked(),
		ap.WithPreferredUsername("johndoe"),
		ap.WithPublished(MockDate.Add(100*time.Hour)),
		ap.WithUpdated(MockDate.Add(400*time.Hour)),
		ap.WithName("Johnathan Doe"),
	)

	object1 := ap.Note(
		ap.WithID(c2sRootIRI.AddPath("objects/1")),
		ap.WithContent("<p>Hello</p><code>FedBOX</code>!</p>\n"),
		ap.WithMediaType("text/html"),
		ap.WithPublished(MockDate.Add(86400*time.Hour)),
		ap.WithUpdated(MockDate.Add(10000*time.Hour)),
		ap.WithAttributedTo(admin1.ID),
		ap.WithSource(vocab.Source{
			Content:   vocab.NaturalLanguageValues{vocab.NilLangRef: vocab.Content("Hello `FedBOX`!")},
			MediaType: "text/markdown",
		}),
		ap.WithTo(vocab.IRI("https://www.w3.org/ns/activitystreams#Public")),
	)

	conf := fedbox.C2SConfig(
		fedbox.WithImageName(imageName),
		fedbox.WithItems(tag0, object1, admin1, actor2),
		fedbox.WithPrivateKey(privateKey),
		fedbox.Verbose(verbose), fedbox.WithCodeCoverage(coverage),
	)
	cont, err := fedbox.StartContainers(t.Context(), t, conf)
	if err != nil {
		t.Fatalf("Error: %s", err)
	}

	toRun := []tests.RunnableTest{
		tests.HTTPTest{
			Name: "service",
			Req:  tests.Request().IRI(c2sRootIRI),
			Res: tests.Response().
				HasCode(http.StatusOK).
				HasContentType(client.ContentTypeJsonLD).
				ItemMatch(
					tests.HasID(service.ID),
					tests.IsType(service.Type),
					tests.HasPublicKey(service.PublicKey),
					tests.HasPreferredUsername(service.PreferredUsername),
					tests.HasAttributedTo(service.AttributedTo),
					tests.WasPublished(time.Now().Round(0)),
					tests.HasContext(service.Context),
					tests.HasAudience(service.Audience),
					tests.HasSummary(service.Summary),
					tests.HasURL(service.URL),
					tests.HasStreams(service.Streams...),
					tests.HasEndpoints(service.Endpoints),
				),
		},
		tests.HTTPTest{
			Name: "service outbox",
			Req: tests.Request().
				IRI(vocab.Outbox.IRI(service)),
			Res: tests.Response().
				HasCode(http.StatusOK).
				HasContentType(client.ContentTypeJsonLD).
				ItemMatch(
					tests.HasID(
						filterIRI(vocab.Outbox.IRI(c2sRootIRI), filters.WithMaxCount(filters.MaxItems)),
					),
				),
		},
		tests.HTTPTest{
			Name: "service inbox",
			Req: tests.Request().
				IRI(vocab.Inbox.IRI(service)),
			Res: tests.Response().
				HasCode(http.StatusOK).
				HasContentType(client.ContentTypeJsonLD).
				ItemMatch(
					tests.HasID(
						filterIRI(vocab.Inbox.IRI(c2sRootIRI), filters.WithMaxCount(filters.MaxItems)),
					),
				),
		},
		tests.HTTPTest{
			Name: "actors/1",
			Req:  tests.Request().IRI(admin1.ID),
			Res: tests.Response().
				HasCode(http.StatusOK).
				HasContentType(client.ContentTypeJsonLD).
				ItemMatch(
					tests.HasID(admin1.ID),
					tests.HasURL(admin1.URL),
				),
		},
		tests.HTTPTest{
			Name: "actors/1 outbox",
			Req: tests.Request().
				IRI(vocab.Outbox.IRI(admin1)),
			Res: tests.Response().
				HasCode(http.StatusOK).
				HasContentType(client.ContentTypeJsonLD).
				ItemMatch(
					tests.HasID(
						filterIRI(vocab.Outbox.IRI(admin1), filters.WithMaxCount(filters.MaxItems)),
					),
				),
		},
		tests.HTTPTest{
			Name: "actors/1 inbox",
			Req: tests.Request().
				IRI(vocab.Inbox.IRI(admin1)),
			Res: tests.Response().
				HasCode(http.StatusOK).
				HasContentType(client.ContentTypeJsonLD).
				ItemMatch(
					tests.HasID(
						filterIRI(vocab.Inbox.IRI(admin1), filters.WithMaxCount(filters.MaxItems)),
					),
				),
		},
		tests.HTTPTest{
			Name: "objects/0",
			Req:  tests.Request().IRI(tag0.ID),
			Res: tests.Response().
				HasCode(http.StatusOK).
				HasContentType(client.ContentTypeJsonLD).
				ItemMatch(
					tests.HasID(tag0.ID),
					tests.IsType(tag0.Type),
					tests.HasName(tag0.Name),
					tests.HasTo(tag0.To...),
				),
		},
		tests.HTTPTest{
			Name: "objects/1",
			Req:  tests.Request().IRI(object1.ID),
			Res: tests.Response().
				HasCode(http.StatusOK).
				HasContentType(client.ContentTypeJsonLD).
				ItemMatch(
					tests.HasID(object1.ID),
					tests.IsType(object1.Type),
					tests.HasContent(object1.Content),
					tests.WasPublished(object1.Published),
					tests.WasUpdated(object1.Updated),
					tests.HasTo(object1.To...),
					tests.HasMediaType(object1.MediaType),
					tests.HasSource(object1.Source),
				),
		},
		tests.HTTPTest{
			Name: "actors/2",
			Req:  tests.Request().IRI(actor2.ID),
			Res: tests.Response().
				HasCode(http.StatusOK).
				ItemMatch(
					tests.HasID(actor2.ID),
					tests.IsType(actor2.Type),
					tests.HasName(actor2.Name),
					tests.HasSummary(actor2.Summary),
					tests.HasContent(actor2.Content),
					tests.HasAttributedTo(actor2.AttributedTo),
					tests.WasPublished(actor2.Published),
					tests.WasUpdated(actor2.Updated),
					tests.HasAudience(actor2.Audience),
				),
		},
		tests.HTTPTest{
			Name: "actors/2 outbox",
			Req: tests.Request().
				IRI(vocab.Outbox.IRI(actor2)),
			Res: tests.Response().
				HasCode(http.StatusOK).
				HasContentType(client.ContentTypeJsonLD).
				ItemMatch(
					tests.HasID(
						filterIRI(vocab.Outbox.IRI(actor2), filters.WithMaxCount(filters.MaxItems)),
					),
				),
		},
		tests.HTTPTest{
			Name: "actors/2 inbox",
			Req: tests.Request().
				IRI(vocab.Inbox.IRI(actor2)),
			Res: tests.Response().
				HasCode(http.StatusOK).
				HasContentType(client.ContentTypeJsonLD).
				ItemMatch(
					tests.HasID(
						filterIRI(vocab.Inbox.IRI(actor2), filters.WithMaxCount(filters.MaxItems)),
					),
				),
		},
	}

	for _, test := range toRun {
		t.Run(test.Label(), test.Fn(t.Context(), cont))
	}
}
