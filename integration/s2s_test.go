//go:build s2s

package integration

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"testing"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/client/c2s"
	c "github.com/go-ap/fedbox/integration/internal/containers"
	"github.com/go-ap/fedbox/integration/internal/containers/fedbox"
	"github.com/go-ap/fedbox/integration/internal/tests"
	ap "github.com/go-ap/fedbox/integration/internal/vocab"
	"golang.org/x/crypto/ed25519"
)

func Test_S2SRequests(t *testing.T) {
	_, c2sPrvKey, _ := ed25519.GenerateKey(rand.Reader)

	c2sTagAdmin := object(c2sRootIRI.AddPath("objects/0"), ap.HasName("#sysop"))
	c2sAdmin := person(c2sRootIRI.AddPath("actors/1"), ap.HasPreferredUsername("admin"), ap.HasTag(c2sTagAdmin))

	s2sPrvKey, _ := rsa.GenerateKey(rand.Reader, 1024)

	s2sTagAdmin := object(s2sRootIRI.AddPath("objects/0"), ap.HasName("#sysop"))
	s2sAdmin := person(vocab.CollectionPath("actors/1").IRI(s2sRootIRI), ap.HasPreferredUsername("admin"), ap.HasTag(s2sTagAdmin))

	images := c.Suite(
		fedbox.New(
			fedbox.WithImageName(fedBOXImageName),
			fedbox.WithConfig(fedbox.ConfigFromBuildInfo(defaultC2SOptions)),
			fedbox.WithArgs([]string{"--bootstrap"}),
			fedbox.WithKey(c2sPrvKey),
			fedbox.WithRootIRI(c2sRootIRI), fedbox.WithPw(rand.Text()[:8]),
			fedbox.WithItems(c2sTagAdmin, c2sAdmin),
			fedbox.WithTestLogger(t, Verbose),
		),
		fedbox.New(
			fedbox.WithImageName(fedBOXImageName),
			fedbox.WithConfig(fedbox.ConfigFromBuildInfo(defaultS2SOptions)),
			fedbox.WithArgs([]string{"--bootstrap"}),
			fedbox.WithKey(s2sPrvKey),
			fedbox.WithRootIRI(s2sRootIRI), fedbox.WithPw(rand.Text()[:8]),
			fedbox.WithItems(s2sTagAdmin, s2sAdmin),
			fedbox.WithTestLogger(t, Verbose),
		),
	)

	ctx := context.Background()
	cont, err := c.Start(ctx, t, images...)
	if err != nil {
		t.Fatalf("Error: %s", err)
	}

	t.Cleanup(func() {
		cont.Cleanup(t)
	})

	token := new(c2s.BearerSigner)
	toRun := []tests.RunnableTest{
		tests.CommandTest{
			Name: "GenKeys admin",
			Host: string(c2sRootIRI),
			Cmd: c.SSHCmd{
				Cmd:  []string{"accounts", "gen-keys"},
				User: string(c2sRootIRI),
				Key:  c2sPrvKey,
			},
			IO: tests.WithTests(tests.EndOK),
		},
		tests.CommandTest{
			Name: "Gen OAuth2 bearer",
			Host: string(c2sRootIRI),
			Cmd: c.SSHCmd{
				Cmd:  []string{"oauth", "token", "add", string(c2sAdmin.ID)},
				User: string(c2sRootIRI),
				Key:  c2sPrvKey,
			},
			IO: tests.WithTests(tests.GetToken(token), tests.EndOK),
		},
		tests.TestSuite{
			Name: "Create Note",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Simple note",
					Req: tests.Request().IRI(c2sAdmin.Outbox.GetLink()).
						Post().
						Signer(token.Sign).
						BodyItem(vocab.Flag{
							Type:  vocab.CreateType,
							Actor: c2sAdmin,
							Object: &vocab.Object{
								Type:    vocab.NoteType,
								To:      vocab.ItemCollection{s2sAdmin.ID},
								Content: vocab.DefaultNaturalLanguage("Lorem Ipsum"),
							},
						}),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						ItemMatch(
							tests.IsType(vocab.NoteType),
							tests.HasContent(vocab.DefaultNaturalLanguage("Lorem Ipsum")),
						),
				},
			},
		},
	}
	for _, test := range toRun {
		t.Run(test.Label(), test.Fn(ctx, cont))
	}
}
