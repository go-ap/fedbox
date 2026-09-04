//go:build c2s

package integration

import (
	"crypto/rand"
	"net/http"
	"testing"
	"time"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/client"
	"github.com/go-ap/fedbox/integration/internal/containers/fedbox"
	"github.com/go-ap/fedbox/integration/internal/tests"
	ap "github.com/go-ap/fedbox/integration/internal/vocab"
	"github.com/go-ap/filters"
	"golang.org/x/crypto/ed25519"
)

func Test_Fetch(t *testing.T) {
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	service := root(c2sRootIRI, ap.HasPublicKey(publicKey), ap.HasTo(vocab.PublicNS), ap.HasCC(c2sRootIRI))

	tag0 := object(
		c2sRootIRI.AddPath("objects/0"),
		ap.HasName("#sysop"),
		ap.HasTo(vocab.PublicNS),
	)

	admin1 := person(c2sRootIRI.AddPath("actors/1"),
		ap.HasPreferredUsername("admin"),
		ap.HasTag(tag0),
	)

	actor2 := person(c2sRootIRI.AddPath("actors/2"),
		ap.HasContent("Generated actor"),
		ap.HasSummary("Generated actor"),
		ap.HasLiked,
		ap.HasPreferredUsername("johndoe"),
		ap.HasPublished(MockDate.Add(100*time.Hour)),
		ap.HasUpdated(MockDate.Add(400*time.Hour)),
		ap.HasName("Johnathan Doe"),
	)

	object1 := object(c2sRootIRI.AddPath("objects/1"),
		ap.HasType(vocab.NoteType),
		ap.HasContent("<p>Hello</p><code>FedBOX</code>!</p>\n"),
		ap.HasMediaType("text/html"),
		ap.HasPublished(MockDate.Add(86400*time.Hour)),
		ap.HasUpdated(MockDate.Add(10000*time.Hour)),
		ap.HasAttributedTo(admin1.ID),
		ap.HasSource("Hello `FedBOX`!", "text/markdown"),
		ap.HasTo("https://www.w3.org/ns/activitystreams#Public"),
	)

	conf := fedbox.C2SConfig(fedBOXImageName, tag0, object1, admin1, actor2, privateKey)
	cont, err := fedbox.StartContainers(t.Context(), t, conf)
	if err != nil {
		t.Fatalf("Error: %s", err)
	}
	t.Cleanup(func() {
		cont.Cleanup(t)
	})

	toRun := []tests.HTTPTest{
		{
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
		{
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
		{
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
		{
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
		{
			Name: "actors/1 outbox",
			Req: tests.Request().
				IRI(admin1.Outbox.GetLink()),
			Res: tests.Response().
				HasCode(http.StatusOK).
				HasContentType(client.ContentTypeJsonLD).
				ItemMatch(
					tests.HasID(
						filterIRI(admin1.Outbox.GetLink(), filters.WithMaxCount(filters.MaxItems)),
					),
				),
		},
		{
			Name: "actors/1 inbox",
			Req: tests.Request().
				IRI(admin1.Inbox.GetLink()),
			Res: tests.Response().
				HasCode(http.StatusOK).
				HasContentType(client.ContentTypeJsonLD).
				ItemMatch(
					tests.HasID(
						filterIRI(admin1.Inbox.GetLink(), filters.WithMaxCount(filters.MaxItems)),
					),
				),
		},
		{
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
		{
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
					tests.HasSource(object1.Source.Content, object1.Source.MediaType),
				),
		},
		{
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
		{
			Name: "actors/2 outbox",
			Req: tests.Request().
				IRI(actor2.Outbox.GetLink()),
			Res: tests.Response().
				HasCode(http.StatusOK).
				HasContentType(client.ContentTypeJsonLD).
				ItemMatch(
					tests.HasID(
						filterIRI(actor2.Outbox.GetLink(), filters.WithMaxCount(filters.MaxItems)),
					),
				),
		},
		{
			Name: "actors/2 inbox",
			Req: tests.Request().
				IRI(actor2.Inbox.GetLink()),
			Res: tests.Response().
				HasCode(http.StatusOK).
				HasContentType(client.ContentTypeJsonLD).
				ItemMatch(
					tests.HasID(
						filterIRI(actor2.Inbox.GetLink(), filters.WithMaxCount(filters.MaxItems)),
					),
				),
		},
	}

	for _, test := range toRun {
		t.Run(test.Name, test.Fn(t.Context(), cont))
	}
}
