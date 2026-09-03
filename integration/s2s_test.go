//go:build s2s

package integration

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"testing"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/client"
	"github.com/go-ap/client/c2s"
	c "github.com/go-ap/fedbox/integration/internal/containers"
	"github.com/go-ap/fedbox/integration/internal/containers/fedbox"
	"github.com/go-ap/fedbox/integration/internal/tests"
	ap "github.com/go-ap/fedbox/integration/internal/vocab"
	"github.com/go-ap/filters"
	"golang.org/x/crypto/ed25519"
)

var (
	s2sPerson1 = person(
		ap.HasID(s2sRootIRI.AddPath("actors/person-1")),
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
		ap.HasSharedInbox(vocab.Inbox.IRI(s2sRootIRI)),
	)

	s2sPerson3 = person(
		s2sRootIRI.AddPath("actors/person-3"),
		ap.HasPreferredUsername("alice"),
		ap.HasContent("lorem ipsum dolor sic amet"),
		ap.HasAudience(vocab.PublicNS),
		ap.HasPublished(MockDate),
		ap.HasFollowing,
		ap.HasFollowers,
		ap.HasShares,
		ap.HasSharedInbox(vocab.Inbox.IRI(s2sRootIRI)),
	)
)

func Test_S2S_SharedInbox(t *testing.T) {
	s2sPrvKey, _ := rsa.GenerateKey(rand.Reader, 1024)
	s2sAdmin := person(vocab.CollectionPath("actors/1").IRI(s2sRootIRI), ap.HasPreferredUsername("admin"))

	verbose := true
	adminTok := new(c2s.BearerSigner)
	rootExec := c.ExecAs(c2sRootIRI, ed2559Key)
	c2sConf := fedbox.C2SConfig(fedBOXImageName, ed2559Key,
		tagAdmin, admin,
		rootExec.ExtractOAuth2Bearer(admin.ID, adminTok),
	)

	// NOTE(marius): s2sAdmin does not have a shared inbox, while s2sPerson1 and s2sPerson3 do
	s2sConf := fedbox.S2SConfig(fedBOXImageName, s2sPrvKey, verbose,
		s2sAdmin, s2sPerson1, s2sPerson3,
	)

	cont, err := fedbox.StartContainers(t.Context(), t, c2sConf, s2sConf)
	if err != nil {
		t.Fatalf("Error: %s", err)
	}

	t.Cleanup(func() {
		cont.Cleanup(t)
	})

	article5 := object(
		c2sRootIRI.AddPath("objects/article-5"),
		ap.HasType(vocab.ArticleType),
		ap.HasContent("lorem ipsum dolor sic amet"),
		ap.HasAudience(vocab.PublicNS),
		ap.HasReplies,
	)
	create4ID := c2sRootIRI.AddPath("activities/create-4")
	create4 := create(
		ap.HasActor(admin),
		ap.HasObject(article5),
		// NOTE(marius): make sure we dispatch to the sharedInbox of s2sPerson1 and s2sPerson3
		//  which is the s2sRootIRI
		ap.HasTo(s2sRootIRI),
	)

	toRun := []tests.RunnableTest{
		tests.TestSuite{
			Name: "pre-checks",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "first instance admin exists",
					Req: tests.Request().
						IRI(admin.ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(admin.ID),
							tests.IsType(admin.Type),
							tests.HasInbox(vocab.Inbox.IRI(admin)),
							tests.HasOutbox(vocab.Outbox.IRI(admin)),
						),
				},
				tests.HTTPTest{
					Name: "second instance admin exists, and does not have a sharedInbox",
					Req: tests.Request().
						IRI(s2sAdmin.ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(s2sAdmin.ID),
							tests.IsType(s2sAdmin.Type),
							tests.HasInbox(vocab.Inbox.IRI(s2sAdmin)),
							tests.HasOutbox(vocab.Outbox.IRI(s2sAdmin)),
							// NOTE(marius): this is nil - so it succeeds
							tests.HasSharedInbox(s2sAdmin.Endpoints.SharedInbox),
						),
				},
				tests.HTTPTest{
					Name: "second instance person1 exists, and has sharedInbox",
					Req: tests.Request().
						IRI(s2sPerson1.ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(s2sPerson1.ID),
							tests.IsType(s2sPerson1.Type),
							tests.HasInbox(vocab.Inbox.IRI(s2sPerson1)),
							tests.HasOutbox(vocab.Outbox.IRI(s2sPerson1)),
							tests.HasSharedInbox(s2sPerson1.Endpoints.SharedInbox),
						),
				},
				tests.HTTPTest{
					Name: "second instance person3 exists, and has sharedInbox",
					Req: tests.Request().
						IRI(s2sPerson3.ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(s2sPerson3.ID),
							tests.IsType(s2sPerson3.Type),
							tests.HasInbox(vocab.Inbox.IRI(s2sPerson3)),
							tests.HasOutbox(vocab.Outbox.IRI(s2sPerson3)),
							tests.HasSharedInbox(s2sPerson3.Endpoints.SharedInbox),
						),
				},
			},
		},

		tests.TestSuite{
			Name: "Create directly to sharedInbox",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Create article",
					Req: tests.Request().
						Bearer(adminTok.AccessToken).
						IRI(vocab.Outbox.IRI(admin)).
						BodyItem(create4),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						HasLocation(create4ID).
						ItemMatch(
							tests.HasID(article5.ID),
							tests.IsType(article5.Type),
							tests.HasContent(article5.Content),
							tests.HasReplies(vocab.Replies.IRI(article5)),
						),
				},
				tests.HTTPTest{
					Name: "Create is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(create4ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(create4ID),
							tests.IsType(create4.Type),
							tests.HasActor(create4.Actor),
							tests.HasObject(create4.Object),
						),
				},
				tests.HTTPTest{
					Name: "Article is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(article5.ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(article5.ID),
							tests.IsType(article5.Type),
							tests.HasContent(article5.Content),
						),
				},
				tests.HTTPTest{
					Name: "Create is in first instance's admin Outbox",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(admin)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Outbox.IRI(admin), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(1),
							tests.HasItem(create4ID),
						),
				},
				tests.HTTPTest{
					Name: "Create is in first instance's root Inbox",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Inbox.IRI(c2sRootIRI)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasID(filterIRI(vocab.Inbox.IRI(c2sRootIRI), filters.WithMaxCount(100))),
							tests.HasTotalItems(4),
							tests.HasItem(create4ID),
						),
				},
				tests.HTTPTest{
					Name: "Create is in second instance's root Inbox",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Inbox.IRI(s2sRootIRI)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasID(filterIRI(vocab.Inbox.IRI(s2sRootIRI), filters.WithMaxCount(100))),
							tests.HasTotalItems(5),
							tests.HasItem(create4ID),
						),
				},
				tests.HTTPTest{
					Name: "Create is NOT in second instance's admin Inbox",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Inbox.IRI(s2sAdmin.ID)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasID(filterIRI(vocab.Inbox.IRI(s2sAdmin), filters.WithMaxCount(100))),
							tests.HasTotalItems(0),
							tests.DoesNotHaveItem(create4ID),
						),
				},
				tests.HTTPTest{
					Name: "Create is in second instance's s2sPerson1 Inbox",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Inbox.IRI(s2sPerson1.ID)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasID(filterIRI(vocab.Inbox.IRI(s2sPerson1), filters.WithMaxCount(100))),
							tests.HasTotalItems(3),
							tests.HasItem(create4ID),
						),
				},
				tests.HTTPTest{
					Name: "Create is in second instance's s2sPerson3 Inbox",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Inbox.IRI(s2sPerson3.ID)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasID(filterIRI(vocab.Inbox.IRI(s2sPerson3), filters.WithMaxCount(100))),
							tests.HasTotalItems(2),
							tests.HasItem(create4ID),
						),
				},
			},
		},
	}

	for _, test := range toRun {
		t.Run(test.Label(), test.Fn(t.Context(), cont))
	}
}

func Test_S2SRequests(t *testing.T) {
	_, c2sPrvKey, _ := ed25519.GenerateKey(rand.Reader)
	c2sTagAdmin := object(c2sRootIRI.AddPath("objects/0"), ap.HasName("#sysop"))
	c2sAdmin := person(c2sRootIRI.AddPath("actors/1"), ap.HasPreferredUsername("admin"), ap.HasTag(c2sTagAdmin))

	s2sPrvKey, _ := rsa.GenerateKey(rand.Reader, 1024)
	s2sTagAdmin := object(s2sRootIRI.AddPath("objects/0"), ap.HasName("#sysop"))
	s2sAdmin := person(vocab.CollectionPath("actors/1").IRI(s2sRootIRI), ap.HasPreferredUsername("admin"), ap.HasTag(s2sTagAdmin))

	c2sConf := fedbox.C2SConfig(fedBOXImageName, c2sTagAdmin, c2sAdmin, c2sPrvKey)
	s2sConf := fedbox.C2SConfig(fedBOXImageName, s2sTagAdmin, s2sAdmin, s2sPrvKey)

	cont, err := fedbox.StartContainers(t.Context(), t, c2sConf, s2sConf)
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
		t.Run(test.Label(), test.Fn(t.Context(), cont))
	}
}
