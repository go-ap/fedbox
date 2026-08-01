//go:build c2s

package integration

import (
	"context"
	"crypto/rand"
	"net/http"
	"testing"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/client"
	c "github.com/go-ap/fedbox/integration/internal/containers"
	"github.com/go-ap/fedbox/integration/internal/containers/fedbox"
	"github.com/go-ap/fedbox/integration/internal/tests"
	ap "github.com/go-ap/fedbox/integration/internal/vocab"
	"golang.org/x/crypto/ed25519"
)

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

	contentType := client.ContentTypeJsonLD
	toRun := []tests.HTTPTest{
		{
			Name: "service",
			Req:  tests.Request().IRI(c2sRootIRI),
			Res:  tests.Response().HasCode(http.StatusOK).HasContentType(contentType).HasExactItem(service),
		},
		{
			Name: "actors/1",
			Req:  tests.Request().IRI(admin1.ID),
			Res:  tests.Response().HasCode(http.StatusOK).HasContentType(contentType).HasExactItem(admin1),
		},
		{
			Name: "objects/0",
			Req:  tests.Request().IRI(tag0.ID),
			Res:  tests.Response().HasCode(http.StatusOK).HasContentType(contentType).HasExactItem(tag0),
		},
		{
			Name: "objects/1",
			Req:  tests.Request().IRI(object1.ID),
			Res:  tests.Response().HasCode(http.StatusOK).HasContentType(contentType).HasExactItem(object1),
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
		fedbox.WithKey(privateKey),
		fedbox.WithPw(rand.Text()[:8]),
		fedbox.WithItems(tag0, object1, admin1, actor2),
	))

	ctx := context.Background()
	cont, err := c.Start(ctx, t, images...)
	if err != nil {
		t.Fatalf("Error: %s", err)
	}

	t.Cleanup(func() {
		cont.Cleanup(t)
	})

	for _, test := range toRun {
		t.Run(test.Name, test.Fn(ctx, cont))
	}
}
