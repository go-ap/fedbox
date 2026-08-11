//go:build c2s

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
	ap2 "github.com/go-ap/fedbox/activitypub"
	c "github.com/go-ap/fedbox/integration/internal/containers"
	"github.com/go-ap/fedbox/integration/internal/containers/fedbox"
	"github.com/go-ap/fedbox/integration/internal/tests"
	ap "github.com/go-ap/fedbox/integration/internal/vocab"
	"github.com/go-ap/filters"
	"golang.org/x/crypto/ed25519"
)

func Test_C2S_CreateRequests(t *testing.T) {
	_, prvKey, _ := ed25519.GenerateKey(rand.Reader)

	admin := person(
		c2sRootIRI.AddPath("actors/1"),
		ap.HasPreferredUsername("admin"),
		ap.HasAudience(vocab.PublicNS),
	)
	token := new(c2s.BearerSigner)
	rootExec := c.ExecAs(c2sRootIRI, prvKey)

	images := c.Suite(fedbox.New(
		fedbox.WithImageName(fedBOXImageName),
		fedbox.WithConfig(fedbox.ConfigFromBuildInfo(defaultC2SOptions)),
		fedbox.WithArgs([]string{"--bootstrap"}),
		fedbox.WithKey(prvKey),
		fedbox.WithTestLogger(t, Verbose),
		fedbox.WithItems(admin),
		fedbox.WithCmd(rootExec.ExtractOAuth2Bearer(admin.ID, token)),
	))

	ctx := context.Background()
	cont, err := c.Start(ctx, t, images...)
	if err != nil {
		t.Fatalf("Unable to start test containers: %v", err)
	}

	t.Cleanup(func() {
		cont.Cleanup(t)
	})

	person4 := person(
		c2sRootIRI.AddPath("actors/person-4"),
		ap.HasPreferredUsername("example1"),
		ap.HasName("Example McSample"),
		ap.HasReplies,
	)

	article5 := object(
		c2sRootIRI.AddPath("objects/article-5"),
		ap.HasType(vocab.ArticleType),
		ap.HasContent("lorem ipsum dolor sic amet"),
		ap.HasAudience(vocab.PublicNS),
		ap.HasReplies,
	)

	article7 := object(
		c2sRootIRI.AddPath("objects/article-7"),
		ap.HasAudience(vocab.PublicNS),
		ap.HasType(vocab.ArticleType),
		ap.HasContent("this should go to multiple reply collections"),
		ap.HasInReplyTo(article5.ID, person4.ID, c2sRootIRI),
	)

	create3ID := c2sRootIRI.AddPath("activities/create-3")
	create3 := create(ap.HasActor(admin), ap.HasObject(person4))

	create4ID := c2sRootIRI.AddPath("activities/create-4")
	create4 := create(ap.HasActor(admin), ap.HasObject(article5))

	create6ID := c2sRootIRI.AddPath("activities/create-5")
	create6 := create(ap.HasActor(admin), ap.HasObject(article7))

	toRun := []tests.RunnableTest{
		tests.HTTPTest{
			Name: "admin Outbox exists",
			Req: tests.Request().
				Bearer(token.AccessToken).
				Accept(client.ContentTypeJsonActivity).
				IRI(vocab.Outbox.IRI(admin)),
			Res: tests.Response().
				HasCode(http.StatusOK).
				ItemMatch(
					tests.IsType(vocab.OrderedCollectionPageType),
					tests.HasID(filterIRI(vocab.Outbox.IRI(admin), filters.WithMaxCount(filters.MaxItems))),
					tests.HasTotalItems(0),
				),
		},
		tests.TestSuite{
			Name: "Create actor",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Create actor",
					Req: tests.Request().
						Bearer(token.AccessToken).
						IRI(vocab.Outbox.IRI(admin)).
						BodyItem(create3),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						HasLocation(create3ID).
						ItemMatch(
							tests.HasID(person4.ID),
							tests.IsType(person4.Type),
							tests.HasPreferredUsername(person4.PreferredUsername),
							tests.HasName(person4.Name),
							tests.HasInbox(vocab.Inbox.IRI(person4)),
							tests.HasOutbox(vocab.Outbox.IRI(person4)),
							tests.HasReplies(vocab.Replies.IRI(person4)),
						),
				},
				tests.HTTPTest{
					Name: "Create-Actor is in Outbox",
					Req: tests.Request().
						Bearer(token.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(admin)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Outbox.IRI(admin), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(1),
							tests.HasItem(create3ID),
						),
				},
				tests.HTTPTest{
					Name: "Create-Actor is in root Inbox",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Inbox.IRI(c2sRootIRI)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasID(filterIRI(vocab.Inbox.IRI(c2sRootIRI), filters.WithMaxCount(100))),
							tests.HasTotalItems(3),
							tests.HasItem(create3ID),
						),
				},
				tests.HTTPTest{
					Name: "Create is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(create3ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(create3ID),
							tests.IsType(create3.Type),
							tests.HasActor(create3.Actor),
							tests.HasObject(create3.Object),
						),
				},
				tests.HTTPTest{
					Name: "Actor is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(person4.ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(person4.ID),
							tests.IsType(person4.Type),
						),
				},
			},
		},
		tests.TestSuite{
			Name: "Create article",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Create article",
					Req: tests.Request().
						Bearer(token.AccessToken).
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
					Name: "Create-Article is in Outbox",
					Req: tests.Request().
						Bearer(token.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(admin)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Outbox.IRI(admin), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(2),
							tests.HasItem(create4ID),
						),
				},
				tests.HTTPTest{
					Name: "Create is in root Inbox",
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
			},
		},
		tests.TestSuite{
			Name: "Create Note with multiple InReplyTo",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Create article",
					Req: tests.Request().
						Bearer(token.AccessToken).
						IRI(vocab.Outbox.IRI(admin)).
						BodyItem(create6),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						HasLocation(create6ID).
						ItemMatch(
							tests.HasID(article7.ID),
							tests.IsType(article7.Type),
							tests.HasContent(article7.Content),
						),
				},
				tests.HTTPTest{
					Name: "Create-Article is in Outbox",
					Req: tests.Request().
						Bearer(token.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(admin)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Outbox.IRI(admin), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(3),
							tests.HasItem(create6ID),
						),
				},
				tests.HTTPTest{
					Name: "Create is in root Inbox",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Inbox.IRI(c2sRootIRI)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasID(filterIRI(vocab.Inbox.IRI(c2sRootIRI), filters.WithMaxCount(100))),
							tests.HasTotalItems(5),
							tests.HasItem(create6ID),
						),
				},
				tests.HTTPTest{
					Name: "Create is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(create6ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(create6ID),
							tests.IsType(create4.Type),
							tests.HasActor(admin.ID),
							tests.HasObject(article7.ID),
						),
				},
				tests.HTTPTest{
					Name: "Article is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(article7.ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(article7.ID),
							tests.IsType(article7.Type),
							tests.HasContent(article7.Content),
							tests.HasInReplyTo(article5.ID, person4.ID, c2sRootIRI),
						),
				},
				tests.HTTPTest{
					Name: "article-5 has article-7 in replies",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Replies.IRI(article5)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Replies.IRI(article5), filters.WithMaxCount(filters.MaxItems))),
							tests.HasTotalItems(1),
							tests.HasItem(article7.ID),
						),
				},
				tests.HTTPTest{
					Name: "person-4 has article-7 in replies",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Replies.IRI(person4)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Replies.IRI(person4), filters.WithMaxCount(filters.MaxItems))),
							tests.HasTotalItems(1),
							tests.HasItem(article7.ID),
						),
				},
				tests.HTTPTest{
					Name: "admin does not have a replies collection",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Replies.IRI(admin)),
					Res: tests.Response().
						HasCode(http.StatusNotFound),
				},
			},
		},
	}

	for _, test := range toRun {
		t.Run(test.Label(), test.Fn(ctx, cont))
	}
}

func Test_C2S_UpdateRequests(t *testing.T) {
	person1 := person(
		ap.HasID(c2sRootIRI.AddPath("actors/person-1")),
		ap.HasPreferredUsername("jdoe"),
		ap.HasName("John Doe"),
		ap.HasAudience(vocab.PublicNS),
		ap.HasPublished(MockDate),
		ap.HasReplies,
	)
	article3 := object(
		c2sRootIRI.AddPath("objects/article-3"),
		ap.HasType(vocab.ArticleType),
		ap.HasContent("lorem ipsum dolor sic amet"),
		ap.HasAudience(vocab.PublicNS),
		ap.HasPublished(MockDate),
	)

	update4ID := c2sRootIRI.AddPath("activities/update-4")
	update4 := update(
		ap.HasCC(vocab.PublicNS),
		ap.HasActor(person1),
		ap.HasObject(person(
			ap.HasID(person1.ID),
			ap.HasPreferredUsername("jdoe"),
			ap.HasName("Jane Doe"),
			ap.HasAudience(vocab.PublicNS),
		)),
	)

	update5ID := c2sRootIRI.AddPath("activities/update-5")
	update5 := update(
		ap.HasActor(person1),
		ap.HasObject(object(
			ap.HasID(article3.ID),
			ap.HasType(vocab.NoteType),
			ap.HasName("update-test"),
			ap.HasSummary("lorem ipsum dolor sic amet"),
			ap.HasContent("updated-content"),
		)),
	)

	update6ID := c2sRootIRI.AddPath("activities/update-6")
	update6 := update(
		ap.HasCC(vocab.PublicNS),
		ap.HasActor(person1),
		ap.HasObject(object(
			ap.HasID(article3.ID),
			ap.HasType(vocab.NoteType),
			ap.HasName("update-test"),
			ap.HasSummary("lorem ipsum dolor sic amet"),
			ap.HasContent("updated-content"),
			ap.HasInReplyTo(person1.ID),
		)),
	)

	_, prvKey, _ := ed25519.GenerateKey(rand.Reader)
	token := new(c2s.BearerSigner)
	rootExec := c.ExecAs(c2sRootIRI, prvKey)

	images := c.Suite(fedbox.New(
		fedbox.WithImageName(fedBOXImageName),
		fedbox.WithConfig(fedbox.ConfigFromBuildInfo(defaultC2SOptions)),
		fedbox.WithArgs([]string{"--bootstrap"}),
		fedbox.WithKey(prvKey),
		fedbox.WithTestLogger(t, Verbose),
		fedbox.WithItems(person1, article3),
		fedbox.WithCmd(rootExec.ExtractOAuth2Bearer(person1.ID, token)),
	))

	ctx := context.Background()
	cont, err := c.Start(ctx, t, images...)
	if err != nil {
		t.Fatalf("Unable to start test containers: %v", err)
	}

	t.Cleanup(func() {
		cont.Cleanup(t)
	})

	toRun := []tests.RunnableTest{
		// NOTE(marius): control checks
		tests.HTTPTest{
			Name: "person exists",
			Req: tests.Request().
				Accept(client.ContentTypeJsonActivity).
				IRI(person1.ID),
			Res: tests.Response().
				HasCode(http.StatusOK).
				ItemMatch(
					tests.HasID(person1.ID),
					tests.IsType(person1.Type),
					tests.HasName(person1.Name),
					tests.HasPreferredUsername(person1.PreferredUsername),
					tests.HasSummary(person1.Summary),
					tests.HasContent(person1.Content),
					tests.WasPublished(person1.Published),
					tests.HasAudience(person1.Audience),
				),
		},
		tests.HTTPTest{
			Name: "article exists",
			Req: tests.Request().
				Accept(client.ContentTypeJsonActivity).
				IRI(article3.ID),
			Res: tests.Response().
				HasCode(http.StatusOK).
				ItemMatch(
					tests.HasID(article3.ID),
					tests.IsType(article3.Type),
					tests.HasName(article3.Name),
					tests.HasSummary(article3.Summary),
					tests.HasContent(article3.Content),
					tests.WasPublished(person1.Published),
				),
		},
		tests.TestSuite{
			Name: "Update actor",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Update actor",
					Req: tests.Request().
						Bearer(token.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1)).
						BodyItem(update4),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						ItemMatch(
							tests.HasID(update4ID),
							tests.IsType(update4.Type),
							tests.HasActor(person1.ID),
							tests.HasObject(person1.ID),
							tests.WasPublished(time.Now().Round(0)),
						),
				},
				tests.HTTPTest{
					Name: "Update is in Outbox",
					Req: tests.Request().
						Bearer(token.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Outbox.IRI(person1), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(1),
							tests.HasItem(update4ID),
						),
				},
				tests.HTTPTest{
					Name: "Update is in root Inbox",
					Req: tests.Request().
						Bearer(token.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Inbox.IRI(c2sRootIRI)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasID(filterIRI(vocab.Inbox.IRI(c2sRootIRI), filters.WithMaxCount(100))),
							tests.HasTotalItems(4),
							tests.HasItem(update4ID),
						),
				},
				tests.HTTPTest{
					Name: "Update is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(update4ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(update4ID),
							tests.IsType(update4.Type),
							tests.HasActor(update4.Actor),
							tests.HasObject(update4.Object),
							tests.WasPublished(time.Now().Round(0)),
						),
				},
				tests.HTTPTest{
					Name: "actor has been updated",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(person1.ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(person1.ID),
							tests.IsType(person1.Type),
							tests.HasName("Jane Doe"),
							tests.HasPreferredUsername(person1.PreferredUsername),
							tests.WasPublished(person1.Published),
							tests.WasUpdated(time.Now().Round(0)),
						),
				},
			},
		},
		tests.TestSuite{
			Name: "Update article",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Update article",
					Req: tests.Request().
						Bearer(token.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1)).
						BodyItem(update5),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						ItemMatch(
							tests.HasID(update5ID),
							tests.IsType(update5.Type),
							tests.HasActor(person1.ID),
							tests.HasObject(article3.ID),
							tests.WasPublished(time.Now().Round(0)),
						),
				},
				tests.HTTPTest{
					Name: "Update is in Outbox",
					Req: tests.Request().
						Bearer(token.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Outbox.IRI(person1), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(2),
							tests.HasItem(update4ID),
							tests.HasItem(update5ID),
						),
				},
				tests.HTTPTest{
					Name: "Update is not in the root Inbox",
					Req: tests.Request().
						Bearer(token.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Inbox.IRI(c2sRootIRI)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasID(filterIRI(vocab.Inbox.IRI(c2sRootIRI), filters.WithMaxCount(100))),
							tests.HasTotalItems(4),
							tests.HasItem(update4ID),
							tests.DoesNotHaveItem(update5ID),
						),
				},
				tests.HTTPTest{
					Name: "Update is not publicly accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(update5ID),
					Res: tests.Response().HasCode(http.StatusNotFound),
				},
				tests.HTTPTest{
					Name: "Update is accessible for person1",
					Req: tests.Request().
						Bearer(token.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(update5ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(update5ID),
							tests.IsType(update5.Type),
							tests.HasActor(update5.Actor),
							tests.HasObject(update5.Object),
							tests.WasPublished(time.Now().Round(0)),
						),
				},
				tests.HTTPTest{
					Name: "article has been updated",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(article3.ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(article3.ID),
							tests.IsType(vocab.NoteType),
							tests.HasName("update-test"),
							tests.HasSummary(article3.Content),
							tests.HasContent("updated-content"),
							tests.WasPublished(article3.Published),
							tests.WasUpdated(time.Now().Round(0)),
						),
				},
			},
		},
		tests.TestSuite{
			Name: "Update note by adding InReplyTo",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "update again",
					Req: tests.Request().
						Bearer(token.AccessToken).
						IRI(vocab.Outbox.IRI(person1)).
						BodyItem(update6),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						ItemMatch(
							tests.HasID(update6ID),
							tests.IsType(update6.Type),
							tests.HasActor(person1.ID),
							tests.HasObject(article3.ID),
							tests.WasPublished(time.Now().Round(0)),
						),
				},
				tests.HTTPTest{
					Name: "update is in Outbox",
					Req: tests.Request().
						Bearer(token.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasID(filterIRI(vocab.Outbox.IRI(person1), filters.WithMaxCount(100))),
							tests.HasTotalItems(3),
							tests.HasItem(update4ID),
							tests.HasItem(update5ID),
							tests.HasItem(update6ID),
						),
				},
				tests.HTTPTest{
					Name: "update is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(update6ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(update6ID),
							tests.IsType(update6.Type),
							tests.HasActor(person1.ID),
							tests.HasObject(article3.ID),
						),
				},
				tests.HTTPTest{
					Name: "note is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(article3.ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(article3.ID),
							tests.IsType(vocab.NoteType),
							tests.HasName("update-test"),
							tests.HasSummary(article3.Content),
							tests.HasContent("updated-content"),
							tests.WasPublished(article3.Published),
							tests.WasUpdated(time.Now().Round(0)),
							tests.HasInReplyTo(person1.ID),
						),
				},
				tests.HTTPTest{
					Name: "person1 has the note in replies",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Replies.IRI(person1.ID)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Replies.IRI(person1.ID), filters.WithMaxCount(filters.MaxItems))),
							tests.HasTotalItems(1),
							tests.HasItem(article3.ID),
						),
				},
			},
		},
	}

	for _, test := range toRun {
		t.Run(test.Label(), test.Fn(ctx, cont))
	}
}

func Test_C2S_DeleteRequests(t *testing.T) {
	person1 := person(
		ap.HasID(c2sRootIRI.AddPath("actors/person-1")),
		ap.HasPreferredUsername("jdoe"),
		ap.HasName("John Doe"),
		ap.HasAudience(vocab.PublicNS),
		ap.HasPublished(MockDate),
		ap.HasReplies,
	)
	article3 := object(
		c2sRootIRI.AddPath("objects/article-3"),
		ap.HasType(vocab.ArticleType),
		ap.HasContent("lorem ipsum dolor sic amet"),
		ap.HasAudience(vocab.PublicNS),
		ap.HasPublished(MockDate),
	)

	_, prvKey, _ := ed25519.GenerateKey(rand.Reader)

	token := new(c2s.BearerSigner)
	rootExec := c.ExecAs(c2sRootIRI, prvKey)

	images := c.Suite(fedbox.New(
		fedbox.WithImageName(fedBOXImageName),
		fedbox.WithConfig(fedbox.ConfigFromBuildInfo(defaultC2SOptions)),
		fedbox.WithArgs([]string{"--bootstrap"}),
		fedbox.WithKey(prvKey),
		fedbox.WithTestLogger(t, Verbose),
		fedbox.WithItems(person1, article3),
		fedbox.WithCmd(rootExec.ExtractOAuth2Bearer(person1.ID, token)),
	))

	ctx := context.Background()
	cont, err := c.Start(ctx, t, images...)
	if err != nil {
		t.Fatalf("Unable to start test containers: %v", err)
	}

	t.Cleanup(func() {
		cont.Cleanup(t)
	})

	delete4ID := c2sRootIRI.AddPath("activities/delete-4")
	delete4 := del(
		ap.HasCC(vocab.PublicNS),
		ap.HasActor(person1),
		ap.HasObject(article3.ID),
	)

	toRun := []tests.RunnableTest{
		tests.TestSuite{
			Name: "Delete",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Delete article",
					Req: tests.Request().
						Bearer(token.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1)).
						BodyItem(delete4),
					Res: tests.Response().
						HasCode(http.StatusGone).
						ItemMatch(
							tests.HasID(delete4ID),
							tests.IsType(delete4.Type),
							tests.HasActor(person1.ID),
							tests.HasObject(article3.ID),
							tests.WasPublished(time.Now().Round(0)),
						),
				},
				tests.HTTPTest{
					Name: "Delete is in Outbox",
					Req: tests.Request().
						Bearer(token.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Outbox.IRI(person1), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(1),
							tests.HasItem(delete4ID),
						),
				},
				tests.HTTPTest{
					Name: "Delete is in root Inbox",
					Req: tests.Request().
						Bearer(token.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Inbox.IRI(c2sRootIRI)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasID(filterIRI(vocab.Inbox.IRI(c2sRootIRI), filters.WithMaxCount(100))),
							tests.HasTotalItems(4),
							tests.HasItem(delete4ID),
						),
				},
				tests.HTTPTest{
					Name: "Delete is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(delete4ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(delete4ID),
							tests.IsType(delete4.Type),
							tests.HasActor(delete4.Actor),
							tests.HasObject(delete4.Object),
							tests.WasPublished(time.Now().Round(0)),
						),
				},
				tests.HTTPTest{
					Name: "article is now a Tombstone",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(article3.ID),
					Res: tests.Response().
						HasCode(http.StatusGone).
						ItemMatch(
							tests.HasID(article3.ID),
							tests.IsType(vocab.TombstoneType),
							tests.WasDeleted(time.Now().Round(0)),
							tests.HasFormerType(article3.Type),
						),
				},
			},
		},
	}

	for _, test := range toRun {
		t.Run(test.Label(), test.Fn(ctx, cont))
	}
}

func Test_C2S_Requests(t *testing.T) {
	publicKey, prvKey, _ := ed25519.GenerateKey(rand.Reader)

	service := root(c2sRootIRI, ap.HasPublicKey(publicKey))

	tagAdmin := object(
		c2sRootIRI.AddPath("objects/0"),
		ap.HasName("#sysop"),
		ap.HasTo(vocab.PublicNS),
	)
	admin := person(
		c2sRootIRI.AddPath("actors/1"),
		ap.HasPreferredUsername("admin"),
		ap.HasTag(tagAdmin),
	)

	draftSig := s2s.New(s2s.WithActor(service, prvKey))
	token := new(c2s.BearerSigner)

	toRun := []tests.RunnableTest{
		tests.HTTPTest{
			Name: "tag admin",
			Req:  tests.Request().IRI(tagAdmin.ID),
			Res: tests.Response().
				HasCode(http.StatusOK).
				HasContentType(client.ContentTypeJsonLD).
				ItemMatch(
					tests.HasID(tagAdmin.ID),
					tests.IsType(tagAdmin.Type),
					tests.HasName(tagAdmin.Name),
					tests.HasTo(tagAdmin.To),
				),
		},
		tests.HTTPTest{
			Name: "admin",
			Req:  tests.Request().IRI(admin.ID),
			Res: tests.Response().
				HasCode(http.StatusOK).
				HasContentType(client.ContentTypeJsonLD).
				ItemMatch(
					tests.HasID(admin.ID),
					tests.IsType(admin.Type),
					tests.HasAttributedTo(admin.ID),
					tests.HasTag(admin.Tag),
					tests.HasPreferredUsername(admin.PreferredUsername),
					tests.HasSharedInbox(admin.Endpoints.SharedInbox),
					tests.HasProxyURL(admin.Endpoints.ProxyURL),
					// NOTE(marius): extra added properties
					tests.HasTo(vocab.ItemCollection{vocab.PublicNS}),
				),
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
					errors.NewBadRequest(errors.BadRequestf("Activity is not valid: invalid activity id"), "Unable to save activity Flag to http://primary.localdomain/actors/1/inbox"),
					errors.BadRequestf("Activity is not valid: invalid activity id"),
				),
		},
		tests.TestSuite{
			Name: "create to outbox",
			Tests: []tests.RunnableTest{
				tests.CommandTest{
					// NOTE(marius): this command generates an activity and increases object count to 2
					Name: "create OAuth2 token",
					Host: string(c2sRootIRI),
					Cmd: c.SSHCmd{
						Cmd:  []string{"oauth", "token", "add", string(admin.ID)},
						User: c2sRootIRI.String(),
						Key:  prvKey,
					},
					IO: tests.WithTests(tests.GetToken(token), tests.AnyOutput),
				},
				tests.HTTPTest{
					Name: "Create",
					Req: tests.Request().IRI(admin.Outbox.GetLink()).
						Post().
						ContentType(client.ContentTypeJsonLD).
						Signer(token.Sign).
						BodyItem(ap2.WrapObjectInCreate(
							vocab.Object{
								Type:      vocab.NoteType,
								Content:   vocab.DefaultNaturalLanguage("test"),
								Published: MockDate,
							}, admin,
						)),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						HasLocation(c2sRootIRI.AddPath("activities/create-4")).
						ItemMatch(
							tests.HasID(c2sRootIRI.AddPath("objects/note-5")),
							tests.IsType(vocab.NoteType),
							tests.HasContent("test"),
							tests.WasPublished(MockDate),
							tests.HasTo(vocab.ItemCollection{vocab.PublicNS}),
						),
				},
				tests.HTTPTest{
					Name: "check activity",
					Req: tests.Request().
						ContentType(client.ContentTypeJsonLD).
						IRI(c2sRootIRI.AddPath("activities/create-4")),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(c2sRootIRI.AddPath("activities/create-4")),
							tests.IsType(vocab.CreateType),
							tests.HasActor(admin.ID),
							tests.HasObject(c2sRootIRI.AddPath("objects/note-5")),
						),
				},
				tests.HTTPTest{
					Name: "check object",
					Req: tests.Request().
						Signer(token.Sign).
						ContentType(client.ContentTypeJsonLD).
						IRI(c2sRootIRI.AddPath("objects/note-5")),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(c2sRootIRI.AddPath("objects/note-5")),
							tests.IsType(vocab.NoteType),
							tests.HasContent("test"),
							tests.WasPublished(MockDate),
						),
				},
			},
		},
		tests.TestSuite{
			Name: "update to outbox",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Update",
					Req: tests.Request().IRI(admin.Outbox.GetLink()).
						Post().
						ContentType(client.ContentTypeJsonLD).
						Signer(token.Sign).
						BodyItem(ap2.WrapObjectInUpdate(
							&vocab.Object{
								ID:      c2sRootIRI.AddPath("objects/note-5"),
								Type:    vocab.NoteType,
								Content: vocab.DefaultNaturalLanguage("updated"),
							}, admin),
						),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						ItemMatch(
							tests.HasID(c2sRootIRI.AddPath("activities/update-6")),
							tests.IsType(vocab.UpdateType),
							tests.WasPublished(time.Now().Round(0)),
							tests.HasTo(vocab.ItemCollection{vocab.PublicNS}),
							tests.HasActor(admin.ID),
							tests.HasObject(c2sRootIRI.AddPath("objects/note-5")),
						),
				},
				tests.HTTPTest{
					Name: "check update activity",
					Req: tests.Request().
						ContentType(client.ContentTypeJsonLD).
						IRI(c2sRootIRI.AddPath("activities/update-6")),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(c2sRootIRI.AddPath("activities/update-6")),
							tests.IsType(vocab.UpdateType),
							tests.WasPublished(time.Now().Round(0)),
							tests.HasTo(vocab.ItemCollection{vocab.PublicNS}),
							tests.HasActor(admin.ID),
							tests.HasObject(c2sRootIRI.AddPath("objects/note-5")),
						),
				},
				tests.HTTPTest{
					Name: "check updated object",
					Req: tests.Request().
						Signer(token.Sign).
						ContentType(client.ContentTypeJsonLD).
						IRI(c2sRootIRI.AddPath("objects/note-5")),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(c2sRootIRI.AddPath("objects/note-5")),
							tests.IsType(vocab.NoteType),
							tests.HasContent("updated"),
							tests.WasPublished(MockDate),
							//tests.WasUpdated(time.Now().Round(0)), // TODO(marius): this doesn't seem to get updated
						),
				},
			},
		},
		tests.TestSuite{
			Name: "Flag activity",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					// NOTE(marius): due to previous tests, the cnt is at 5, so our activity is flag-5
					Name: "flag to outbox",
					Req: tests.Request().IRI(admin.Outbox.GetLink()).
						Post().
						ContentType(client.ContentTypeJsonLD).
						Signer(token.Sign).
						BodyItem(&vocab.Activity{
							Type:      vocab.FlagType,
							Actor:     admin.ID,
							Object:    admin.ID,
							Published: MockDate,
						}),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						ItemMatch(
							tests.HasID(c2sRootIRI.AddPath("activities/flag-7")),
							tests.IsType(vocab.FlagType),
							tests.HasActor(admin.ID),
							tests.HasObject(admin.ID),
							tests.WasPublished(MockDate),
						),
				},
				tests.HTTPTest{
					Name: "check flag activity",
					Req: tests.Request().
						Bearer(token.AccessToken).
						ContentType(client.ContentTypeJsonLD).
						IRI(c2sRootIRI.AddPath("activities/flag-7")),
					// TODO(marius): check if this is due to the actor that fetches the flag
					//  is the same actor that has been flagged.
					Res: tests.Response().
						HasCode(http.StatusNotFound).
						HasContentType(client.ContentTypeJson).
						HasErrors(errors.NotFoundf("http://primary.localdomain/activities/flag-7 was not found")),
					//Res: tests.Response().
					//	HasCode(http.StatusOK).
					//	ItemMatch(
					//		tests.HasID(c2sRootIRI.AddPath("activities/update-6")),
					//		tests.IsType(vocab.UpdateType),
					//		tests.WasPublished(time.Now().Round(0)),
					//		tests.HasTo(vocab.ItemCollection{vocab.PublicNS}),
					//		tests.HasActor(admin.ID),
					//		tests.HasObject(admin.ID),
					//	),
				},
			},
		},
	}

	images := c.Suite(fedbox.New(
		fedbox.WithImageName(fedBOXImageName),
		fedbox.WithConfig(fedbox.ConfigFromBuildInfo(defaultC2SOptions)),
		fedbox.WithArgs([]string{"--bootstrap"}),
		fedbox.WithKey(prvKey),
		fedbox.WithItems(tagAdmin, admin),
		fedbox.WithTestLogger(t, Verbose),
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
		t.Run(test.Label(), test.Fn(ctx, cont))
	}
}
