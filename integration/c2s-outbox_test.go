//go:build c2s

package integration

import (
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

var (
	_, ed2559Key, _ = ed25519.GenerateKey(rand.Reader)

	tagAdmin = object(
		c2sRootIRI.AddPath("objects/0"),
		ap.HasName("#sysop"),
		ap.HasTo(vocab.PublicNS),
	)
	admin = person(
		c2sRootIRI.AddPath("actors/1"),
		ap.HasPreferredUsername("admin"),
		ap.HasAudience(vocab.PublicNS),
		ap.HasTag(tagAdmin),
	)
	person1 = person(
		ap.HasID(c2sRootIRI.AddPath("actors/person-1")),
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
	)
	person3 = person(
		c2sRootIRI.AddPath("actors/person-3"),
		ap.HasPreferredUsername("alice"),
		ap.HasContent("lorem ipsum dolor sic amet"),
		ap.HasAudience(vocab.PublicNS),
		ap.HasPublished(MockDate),
		ap.HasFollowing,
		ap.HasFollowers,
		ap.HasShares,
	)
)

func Test_C2S_CreateRequests(t *testing.T) {
	token := new(c2s.BearerSigner)
	rootExec := c.ExecAs(c2sRootIRI, ed2559Key)
	conf := fedbox.C2SConfig(fedBOXImageName, admin, ed2559Key, rootExec.ExtractOAuth2Bearer(admin.ID, token))

	ctx := t.Context()
	cont, err := fedbox.StartContainers(ctx, t, conf)
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

	create5ID := c2sRootIRI.AddPath("activities/create-5")
	create5 := create(ap.HasActor(admin), ap.HasObject(article7))

	undo6ID := c2sRootIRI.AddPath("/activities/undo-6")
	undo6 := undo(ap.HasActor(admin), ap.HasObject(create5ID), ap.HasCC(person4.ID))

	undo7ID := c2sRootIRI.AddPath("/activities/undo-7")
	undo7 := undo(ap.HasActor(admin), ap.HasObject(create4ID), ap.HasCC(person4.ID))

	undo8ID := c2sRootIRI.AddPath("/activities/undo-8")
	undo8 := undo(ap.HasActor(admin), ap.HasObject(create3ID))

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
						BodyItem(create5),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						HasLocation(create5ID).
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
							tests.HasItem(create5ID),
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
							tests.HasItem(create5ID),
						),
				},
				tests.HTTPTest{
					Name: "Create is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(create5ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(create5ID),
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
		tests.TestSuite{
			Name: "Undo Create Note with multiple InReplyTo",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Undo Create",
					Req: tests.Request().
						IRI(vocab.Outbox.IRI(admin)).
						Post().
						ContentType(client.ContentTypeJsonLD).
						Signer(token.Sign).
						BodyItem(undo6),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						ItemMatch(
							tests.HasID(undo6ID),
							tests.IsType(undo6.Type),
							tests.HasCC(undo6.CC),
							tests.HasActor(undo6.Actor),
							tests.HasObject(undo6.Object),
							tests.WasPublished(time.Now()),
						),
				},
				tests.HTTPTest{
					Name: "Undo is in admin's outbox, but no Create",
					Req: tests.Request().
						Signer(token.Sign).
						ContentType(client.ContentTypeJsonLD).
						IRI(vocab.Outbox.IRI(admin)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Outbox.IRI(admin), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(3),
							tests.HasItem(undo6ID),
							tests.DoesNotHaveItem(create5ID),
						),
				},
				tests.HTTPTest{
					Name: "Undo is in person4's inbox, but no Create",
					Req: tests.Request().
						Signer(token.Sign).
						ContentType(client.ContentTypeJsonLD).
						IRI(vocab.Inbox.IRI(person4)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Inbox.IRI(person4), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(1),
							tests.HasItem(undo6ID),
							tests.DoesNotHaveItem(create5ID),
						),
				},
				tests.HTTPTest{
					Name: "Create is no longer accessible",
					Req: tests.Request().
						ContentType(client.ContentTypeJson).
						IRI(create5ID),
					Res: tests.Response().
						HasCode(http.StatusNotFound).
						HasErrors(errFedBOXNotFound(create5ID)),
				},
				tests.HTTPTest{
					Name: "Article is no longer accessible",
					Req: tests.Request().
						ContentType(client.ContentTypeJson).
						IRI(article7.ID),
					Res: tests.Response().
						HasCode(http.StatusNotFound).
						HasErrors(errFedBOXNotFound(article7.ID)),
				},
				tests.HTTPTest{
					Name: "article-5 no longer has article-7 in replies",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Replies.IRI(article5)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Replies.IRI(article5), filters.WithMaxCount(filters.MaxItems))),
							tests.HasTotalItems(0),
							tests.DoesNotHaveItem(article7.ID),
						),
				},
				tests.HTTPTest{
					Name: "person-4 no longer has article-7 in replies",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Replies.IRI(person4)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Replies.IRI(person4), filters.WithMaxCount(filters.MaxItems))),
							tests.HasTotalItems(0),
							tests.DoesNotHaveItem(article7.ID),
						),
				},
			},
		},
		tests.TestSuite{
			Name: "Undo Create article",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Undo Create",
					Req: tests.Request().
						IRI(vocab.Outbox.IRI(admin)).
						Post().
						ContentType(client.ContentTypeJsonLD).
						Signer(token.Sign).
						BodyItem(undo7),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						ItemMatch(
							tests.HasID(undo7ID),
							tests.IsType(undo7.Type),
							tests.HasCC(undo7.CC),
							tests.HasActor(undo7.Actor),
							tests.HasObject(undo7.Object),
							tests.WasPublished(time.Now()),
						),
				},
				tests.HTTPTest{
					Name: "Undo is in admin's outbox, but no Create",
					Req: tests.Request().
						Signer(token.Sign).
						ContentType(client.ContentTypeJsonLD).
						IRI(vocab.Outbox.IRI(admin)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Outbox.IRI(admin), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(3),
							tests.HasItem(undo6ID),
							tests.HasItem(undo7ID),
							tests.DoesNotHaveItem(create5ID),
							tests.DoesNotHaveItem(create4ID),
						),
				},
				tests.HTTPTest{
					Name: "Undo is in person4's inbox, but no Create",
					Req: tests.Request().
						Signer(token.Sign).
						ContentType(client.ContentTypeJsonLD).
						IRI(vocab.Inbox.IRI(person4)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Inbox.IRI(person4), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(2),
							tests.HasItem(undo6ID),
							tests.HasItem(undo7ID),
							tests.DoesNotHaveItem(create5ID),
							tests.DoesNotHaveItem(create4ID),
						),
				},
				tests.HTTPTest{
					Name: "Create is no longer accessible",
					Req: tests.Request().
						ContentType(client.ContentTypeJson).
						IRI(create5ID),
					Res: tests.Response().
						HasCode(http.StatusNotFound).
						HasErrors(errFedBOXNotFound(create5ID)),
				},
				tests.HTTPTest{
					Name: "Article is no longer accessible",
					Req: tests.Request().
						ContentType(client.ContentTypeJson).
						IRI(article5.ID),
					Res: tests.Response().
						HasCode(http.StatusNotFound).
						HasErrors(errFedBOXNotFound(article5.ID)),
				},
			},
		},

		tests.TestSuite{
			Name: "Undo Create actor",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Undo Create",
					Req: tests.Request().
						IRI(vocab.Outbox.IRI(admin)).
						Post().
						ContentType(client.ContentTypeJsonLD).
						Signer(token.Sign).
						BodyItem(undo8),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						ItemMatch(
							tests.HasID(undo8ID),
							tests.IsType(undo8.Type),
							tests.HasCC(undo8.CC),
							tests.HasActor(undo8.Actor),
							tests.HasObject(undo8.Object),
							tests.WasPublished(time.Now()),
						),
				},
				tests.HTTPTest{
					Name: "Undo is in admin's outbox, but no Create",
					Req: tests.Request().
						Signer(token.Sign).
						ContentType(client.ContentTypeJsonLD).
						IRI(vocab.Outbox.IRI(admin)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Outbox.IRI(admin), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(3),
							tests.HasItem(undo6ID),
							tests.HasItem(undo7ID),
							tests.HasItem(undo8ID),
							tests.DoesNotHaveItem(create5ID),
							tests.DoesNotHaveItem(create4ID),
							tests.DoesNotHaveItem(create3ID),
						),
				},
				tests.HTTPTest{
					Name: "Create is no longer accessible",
					Req: tests.Request().
						ContentType(client.ContentTypeJson).
						IRI(create5ID),
					Res: tests.Response().
						HasCode(http.StatusNotFound).
						HasErrors(errFedBOXNotFound(create5ID)),
				},
				tests.HTTPTest{
					Name: "Actor is no longer accessible",
					Req: tests.Request().
						ContentType(client.ContentTypeJson).
						IRI(person4.ID),
					Res: tests.Response().
						HasCode(http.StatusNotFound).
						HasErrors(errFedBOXNotFound(person4.ID)),
				},
				tests.HTTPTest{
					Name: "Actor's outbox is no longer accessible",
					Req: tests.Request().
						ContentType(client.ContentTypeJson).
						IRI(vocab.Outbox.IRI(person4.ID)),
					Res: tests.Response().
						HasCode(http.StatusNotFound).
						HasErrors(errFedBOXNotFound(vocab.Outbox.IRI(person4.ID))),
				},
			},
		},
	}

	for _, test := range toRun {
		t.Run(test.Label(), test.Fn(ctx, cont))
	}
}

func Test_C2S_UpdateRequests(t *testing.T) {
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

	token := new(c2s.BearerSigner)
	rootExec := c.ExecAs(c2sRootIRI, ed2559Key)
	conf := fedbox.C2SConfig(fedBOXImageName, person1, article3, ed2559Key, rootExec.ExtractOAuth2Bearer(person1.ID, token))

	cont, err := fedbox.StartContainers(t.Context(), t, conf)
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
		t.Run(test.Label(), test.Fn(t.Context(), cont))
	}
}

func Test_C2S_DeleteRequests(t *testing.T) {
	article3 := object(
		c2sRootIRI.AddPath("objects/article-3"),
		ap.HasType(vocab.ArticleType),
		ap.HasContent("lorem ipsum dolor sic amet"),
		ap.HasAudience(vocab.PublicNS),
		ap.HasPublished(MockDate),
	)

	token := new(c2s.BearerSigner)
	rootExec := c.ExecAs(c2sRootIRI, ed2559Key)
	conf := fedbox.C2SConfig(fedBOXImageName, person1, article3, ed2559Key, rootExec.ExtractOAuth2Bearer(person1.ID, token))

	cont, err := fedbox.StartContainers(t.Context(), t, conf)
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

	delete5ID := c2sRootIRI.AddPath("activities/delete-5")
	delete5 := del(
		ap.HasCC(vocab.PublicNS),
		ap.HasActor(person1),
		ap.HasObject(person1.ID),
	)

	toRun := []tests.RunnableTest{
		tests.TestSuite{
			Name: "Delete article",
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
		tests.TestSuite{
			Name: "Delete actor",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Delete actor",
					Req: tests.Request().
						Bearer(token.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1)).
						BodyItem(delete5),
					Res: tests.Response().
						HasCode(http.StatusGone).
						ItemMatch(
							tests.HasID(delete5ID),
							tests.IsType(delete5.Type),
							tests.HasActor(person1.ID),
							tests.HasObject(person1.ID),
							tests.WasPublished(time.Now().Round(0)),
						),
				},
				tests.HTTPTest{
					Name: "Outbox is no longer accessible",
					Req: tests.Request().
						Bearer(token.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1)),
					Res: tests.Response().
						HasCode(http.StatusNotFound).
						HasErrors(errFedBOXNotFound(vocab.Outbox.IRI(person1))),
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
							tests.HasTotalItems(5),
							tests.HasItem(delete4ID),
							tests.HasItem(delete5ID),
						),
				},
				tests.HTTPTest{
					Name: "Delete is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(delete5ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(delete5ID),
							tests.IsType(delete5.Type),
							tests.HasActor(delete5.Actor),
							tests.HasObject(delete5.Object),
							tests.WasPublished(time.Now().Round(0)),
						),
				},
				tests.HTTPTest{
					Name: "person is now a Tombstone",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(person1.ID),
					Res: tests.Response().
						HasCode(http.StatusGone).
						ItemMatch(
							tests.HasID(person1.ID),
							tests.IsType(vocab.TombstoneType),
							tests.WasDeleted(time.Now().Round(0)),
							tests.HasFormerType(person1.Type),
						),
				},
			},
		},
	}

	for _, test := range toRun {
		t.Run(test.Label(), test.Fn(t.Context(), cont))
	}
}

func Test_C2S_LikeRequests(t *testing.T) {
	article3 := object(
		c2sRootIRI.AddPath("objects/article-3"),
		ap.HasType(vocab.ArticleType),
		ap.HasContent("lorem ipsum dolor sic amet"),
		ap.HasLikes,
		ap.HasAudience(vocab.PublicNS),
		ap.HasPublished(MockDate),
	)

	token := new(c2s.BearerSigner)
	rootExec := c.ExecAs(c2sRootIRI, ed2559Key)
	conf := fedbox.C2SConfig(fedBOXImageName, person1, article3, ed2559Key, rootExec.ExtractOAuth2Bearer(person1.ID, token))

	cont, err := fedbox.StartContainers(t.Context(), t, conf)
	if err != nil {
		t.Fatalf("Unable to start test containers: %v", err)
	}

	t.Cleanup(func() {
		cont.Cleanup(t)
	})

	like4ID := c2sRootIRI.AddPath("activities/like-4")
	like4 := like(
		ap.HasCC(vocab.PublicNS),
		ap.HasActor(person1),
		ap.HasObject(article3.ID),
	)

	like5ID := c2sRootIRI.AddPath("activities/dislike-5")
	like5 := like(
		ap.HasType(vocab.DislikeType),
		ap.HasCC(vocab.PublicNS),
		ap.HasActor(person1),
		ap.HasObject(person1.ID),
	)

	toRun := []tests.RunnableTest{
		tests.TestSuite{
			Name: "Like article",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Like article",
					Req: tests.Request().
						Bearer(token.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1)).
						BodyItem(like4),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						ItemMatch(
							tests.HasID(like4ID),
							tests.IsType(like4.Type),
							tests.HasActor(person1.ID),
							tests.HasObject(article3.ID),
							tests.WasPublished(time.Now().Round(0)),
						),
				},
				tests.HTTPTest{
					Name: "Like is in Outbox",
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
							tests.HasItem(like4ID),
						),
				},
				tests.HTTPTest{
					Name: "Like is in root Inbox",
					Req: tests.Request().
						Bearer(token.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Inbox.IRI(c2sRootIRI)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Inbox.IRI(c2sRootIRI), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(4),
							tests.HasItem(like4ID),
						),
				},
				tests.HTTPTest{
					Name: "Like is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(like4ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(like4ID),
							tests.IsType(like4.Type),
							tests.HasActor(person1.ID),
							tests.HasObject(article3.ID),
							tests.WasPublished(time.Now().Round(0)),
						),
				},
				tests.HTTPTest{
					Name: "Article has a like",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Likes.IRI(article3.ID)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Likes.IRI(article3), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(1),
							tests.HasItem(like4ID),
						),
				},
				tests.HTTPTest{
					Name: "person has article in the liked collection",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Liked.IRI(person1.ID)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Liked.IRI(person1), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(1),
							tests.HasItem(article3.ID),
						),
				},
			},
		},
		tests.TestSuite{
			Name: "Dislike actor",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Dislike actor",
					Req: tests.Request().
						Bearer(token.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1)).
						BodyItem(like5),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						ItemMatch(
							tests.HasID(like5ID),
							tests.IsType(like5.Type),
							tests.HasActor(person1.ID),
							tests.HasObject(person1.ID),
							tests.WasPublished(time.Now().Round(0)),
						),
				},
				tests.HTTPTest{
					Name: "Dislike is in Outbox",
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
							tests.HasItem(like4ID),
							tests.HasItem(like5ID),
						),
				},
				tests.HTTPTest{
					Name: "Dislike is in root Inbox",
					Req: tests.Request().
						Bearer(token.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Inbox.IRI(c2sRootIRI)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasID(filterIRI(vocab.Inbox.IRI(c2sRootIRI), filters.WithMaxCount(100))),
							tests.HasTotalItems(5),
							tests.HasItem(like4ID),
							tests.HasItem(like5ID),
						),
				},
				tests.HTTPTest{
					Name: "Dislike is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(like5ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(like5ID),
							tests.IsType(like5.Type),
							tests.HasActor(like5.Actor),
							tests.HasObject(like5.Object),
							tests.WasPublished(time.Now().Round(0)),
						),
				},
				tests.HTTPTest{
					Name: "person does not have a Dislike",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Likes.IRI(person1.ID)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Likes.IRI(person1), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(0),
							tests.DoesNotHaveItem(like5ID),
						),
				},
				tests.HTTPTest{
					Name: "person does not have itself in the liked collection",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Liked.IRI(person1.ID)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Liked.IRI(person1), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(1),
							tests.HasItem(article3.ID),
							tests.DoesNotHaveItem(person1.ID),
						),
				},
			},
		},
	}

	for _, test := range toRun {
		t.Run(test.Label(), test.Fn(t.Context(), cont))
	}
}

func Test_C2S_ShareRequests(t *testing.T) {
	article3 := object(
		c2sRootIRI.AddPath("objects/article-3"),
		ap.HasType(vocab.ArticleType),
		ap.HasContent("lorem ipsum dolor sic amet"),
		ap.HasAudience(vocab.PublicNS),
		ap.HasPublished(MockDate),
		ap.HasShares,
	)

	token := new(c2s.BearerSigner)
	rootExec := c.ExecAs(c2sRootIRI, ed2559Key)
	conf := fedbox.C2SConfig(fedBOXImageName, person1, article3, ed2559Key, rootExec.ExtractOAuth2Bearer(person1.ID, token))

	cont, err := fedbox.StartContainers(t.Context(), t, conf)
	if err != nil {
		t.Fatalf("Unable to start test containers: %v", err)
	}

	t.Cleanup(func() {
		cont.Cleanup(t)
	})

	announce4ID := c2sRootIRI.AddPath("activities/announce-4")
	announce4 := announce(
		ap.HasCC(vocab.PublicNS),
		ap.HasActor(person1),
		ap.HasObject(article3.ID),
	)

	toRun := []tests.RunnableTest{
		tests.TestSuite{
			Name: "Announce article",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Announce article",
					Req: tests.Request().
						Bearer(token.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1)).
						BodyItem(announce4),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						ItemMatch(
							tests.HasID(announce4ID),
							tests.IsType(announce4.Type),
							tests.HasActor(person1.ID),
							tests.HasObject(article3.ID),
							tests.WasPublished(time.Now().Round(0)),
						),
				},
				tests.HTTPTest{
					Name: "Announce is in Outbox",
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
							tests.HasItem(announce4ID),
						),
				},
				tests.HTTPTest{
					Name: "Announce is in root Inbox",
					Req: tests.Request().
						Bearer(token.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Inbox.IRI(c2sRootIRI)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Inbox.IRI(c2sRootIRI), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(4),
							tests.HasItem(announce4ID),
						),
				},
				tests.HTTPTest{
					Name: "Announce is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(announce4ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(announce4ID),
							tests.IsType(announce4.Type),
							tests.HasActor(person1.ID),
							tests.HasObject(article3.ID),
							tests.WasPublished(time.Now().Round(0)),
						),
				},
				tests.HTTPTest{
					Name: "Article has an Announce",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Shares.IRI(article3.ID)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Shares.IRI(article3), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(1),
							tests.HasItem(announce4ID),
						),
				},
			},
		},
	}

	for _, test := range toRun {
		t.Run(test.Label(), test.Fn(t.Context(), cont))
	}
}

func Test_C2S_FollowRequests(t *testing.T) {
	tokenP1 := new(c2s.BearerSigner)
	tokenP3 := new(c2s.BearerSigner)

	rootExec := c.ExecAs(c2sRootIRI, ed2559Key)
	conf := fedbox.C2SConfig(
		fedBOXImageName,
		person1, person3, ed2559Key,
		rootExec.ExtractOAuth2Bearer(person1.ID, tokenP1),
		rootExec.ExtractOAuth2Bearer(person3.ID, tokenP3),
	)

	cont, err := fedbox.StartContainers(t.Context(), t, conf)
	if err != nil {
		t.Fatalf("Unable to start test containers: %v", err)
	}

	t.Cleanup(func() {
		cont.Cleanup(t)
	})

	follow4ID := c2sRootIRI.AddPath("activities/follow-4")
	follow4 := follow(
		//ap.HasTo(person3.ID), // NOTE(marius): this is added by processing the Follow
		ap.HasCC(vocab.PublicNS),
		ap.HasActor(person1.ID),
		ap.HasObject(person3.ID),
	)
	accept5ID := c2sRootIRI.AddPath("activities/accept-5")
	accept5 := accept(
		ap.HasTo(person1.ID),
		ap.HasCC(vocab.PublicNS),
		ap.HasActor(person3.ID),
		ap.HasObject(follow4ID),
	)

	undo6ID := c2sRootIRI.AddPath("/activities/undo-6")
	undo6 := undo(ap.HasActor(person1), ap.HasObject(follow4ID), ap.HasCC(person3.ID))

	toRun := []tests.RunnableTest{
		tests.TestSuite{
			Name: "control checks",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "person1 exists",
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
					Name: "person1 follower collection exists",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Followers.IRI(person1.ID)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Followers.IRI(person1.ID), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(0),
						),
				},
				tests.HTTPTest{
					Name: "person1 following collection exists",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Following.IRI(person1.ID)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Following.IRI(person1.ID), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(0),
						),
				},
				tests.HTTPTest{
					Name: "person3 exists",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(person3.ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(person3.ID),
							tests.IsType(person3.Type),
							tests.HasName(person3.Name),
							tests.HasPreferredUsername(person3.PreferredUsername),
							tests.HasSummary(person3.Summary),
							tests.HasContent(person3.Content),
							tests.WasPublished(person3.Published),
							tests.HasAudience(person3.Audience),
						),
				},
				tests.HTTPTest{
					Name: "person3 follower collection exists",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Followers.IRI(person3.ID)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Followers.IRI(person3.ID), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(0),
						),
				},
				tests.HTTPTest{
					Name: "person3 following collection exists",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Following.IRI(person3.ID)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Following.IRI(person3.ID), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(0),
						),
				},
			},
		},
		tests.TestSuite{
			Name: "Follow then Accept",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Follow person3",
					Req: tests.Request().
						Bearer(tokenP1.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1)).
						BodyItem(follow4),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						ItemMatch(
							tests.HasID(follow4ID),
							tests.IsType(follow4.Type),
							tests.HasActor(person1.ID),
							tests.HasObject(person3.ID),
							tests.WasPublished(time.Now().Round(0)),
						),
				},
				tests.HTTPTest{
					Name: "Follow is in Outbox",
					Req: tests.Request().
						Bearer(tokenP1.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Outbox.IRI(person1), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(1),
							tests.HasItem(follow4ID),
						),
				},
				tests.HTTPTest{
					Name: "Follow is in root Inbox",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Inbox.IRI(c2sRootIRI)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Inbox.IRI(c2sRootIRI), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(4),
							tests.HasItem(follow4ID),
						),
				},
				tests.HTTPTest{
					Name: "Follow is in person3 Inbox",
					Req: tests.Request().
						Bearer(tokenP3.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Inbox.IRI(person3.ID)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Inbox.IRI(person3.ID), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(1),
							tests.HasItem(follow4ID),
						),
				},
				tests.HTTPTest{
					Name: "Follow is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(follow4ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(follow4ID),
							tests.IsType(follow4.Type),
							tests.HasActor(person1.ID),
							tests.HasObject(person3.ID),
							tests.WasPublished(time.Now().Round(0)),
						),
				},
				tests.HTTPTest{
					Name: "person1 Following is empty",
					Req: tests.Request().
						Bearer(tokenP1.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Following.IRI(person1.ID)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Following.IRI(person1.ID), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(0),
						),
				},
				tests.HTTPTest{
					Name: "person3 Followers is empty",
					Req: tests.Request().
						Bearer(tokenP3.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Followers.IRI(person3.ID)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Followers.IRI(person3.ID), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(0),
						),
				},
				tests.HTTPTest{
					Name: "Accept follow-4",
					Req: tests.Request().
						Bearer(tokenP3.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person3)).
						BodyItem(accept5),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						ItemMatch(
							tests.HasID(accept5ID),
							tests.IsType(accept5.Type),
							tests.HasActor(person3.ID),
							tests.HasTo(person1.ID),
							tests.HasObject(follow4ID),
							tests.WasPublished(time.Now().Round(0)),
						),
				},
				tests.HTTPTest{
					Name: "Accept is in person3's Outbox",
					Req: tests.Request().
						Bearer(tokenP3.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person3)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Outbox.IRI(person3), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(1),
							tests.HasItem(accept5ID),
						),
				},
				tests.HTTPTest{
					Name: "Accept is in root Inbox",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Inbox.IRI(c2sRootIRI)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Inbox.IRI(c2sRootIRI), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(5),
							tests.HasItem(follow4ID),
							tests.HasItem(accept5ID),
						),
				},
				tests.HTTPTest{
					Name: "Accept is in person1's Inbox",
					Req: tests.Request().
						Bearer(tokenP3.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Inbox.IRI(person1.ID)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Inbox.IRI(person1.ID), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(1),
							tests.HasItem(accept5ID),
						),
				},
				tests.HTTPTest{
					Name: "Accept is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(accept5ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(accept5ID),
							tests.IsType(accept5.Type),
							tests.HasActor(person3.ID),
							tests.HasObject(follow4ID),
							tests.HasTo(person1.ID),
							tests.WasPublished(time.Now().Round(0)),
						),
				},
				tests.HTTPTest{
					Name: "person1 Following has person3",
					Req: tests.Request().
						Bearer(tokenP1.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Following.IRI(person1.ID)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Following.IRI(person1.ID), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(1),
							tests.HasItem(person3.ID),
						),
				},
				tests.HTTPTest{
					Name: "person3 Followers has person1",
					Req: tests.Request().
						Bearer(tokenP3.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Followers.IRI(person3.ID)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Followers.IRI(person3.ID), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(1),
							tests.HasItem(person1.ID),
						),
				},
			},
		},

		tests.TestSuite{
			Name: "Undo Follow",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Undo Follow",
					Req: tests.Request().
						IRI(vocab.Outbox.IRI(person1)).
						Post().
						ContentType(client.ContentTypeJsonLD).
						Signer(tokenP1.Sign).
						BodyItem(undo6),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						ItemMatch(
							tests.HasID(undo6ID),
							tests.IsType(undo6.Type),
							tests.HasCC(undo6.CC),
							tests.HasContent(undo6.Content),
							tests.WasPublished(time.Now()),
						),
				},
				tests.HTTPTest{
					Name: "Undo is in person1's outbox, but no Follow",
					Req: tests.Request().
						Signer(tokenP1.Sign).
						ContentType(client.ContentTypeJsonLD).
						IRI(vocab.Outbox.IRI(person1)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Outbox.IRI(person1), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(1),
							tests.HasItem(undo6ID),
							tests.DoesNotHaveItem(follow4ID),
						),
				},
				tests.HTTPTest{
					Name: "Follow is no longer accessible",
					Req: tests.Request().
						ContentType(client.ContentTypeJson).
						IRI(follow4ID),
					Res: tests.Response().
						HasCode(http.StatusNotFound).
						HasErrors(errFedBOXNotFound(follow4ID)),
				},
			},
		},
	}

	for _, test := range toRun {
		t.Run(test.Label(), test.Fn(t.Context(), cont))
	}
}

func Test_C2S_BlockRequests(t *testing.T) {
	tokenP1 := new(c2s.BearerSigner)
	tokenP3 := new(c2s.BearerSigner)

	rootExec := c.ExecAs(c2sRootIRI, ed2559Key)
	conf := fedbox.C2SConfig(
		fedBOXImageName,
		person1, person3, ed2559Key,
		rootExec.ExtractOAuth2Bearer(person1.ID, tokenP1),
		rootExec.ExtractOAuth2Bearer(person3.ID, tokenP3),
	)
	cont, err := fedbox.StartContainers(t.Context(), t, conf)
	if err != nil {
		t.Fatalf("Unable to start test containers: %v", err)
	}

	t.Cleanup(func() {
		cont.Cleanup(t)
	})

	blocked := vocab.CollectionPath("blocked")

	block4ID := c2sRootIRI.AddPath("/activities/block-4")
	block4 := block(
		ap.HasActor(person1.ID),
		ap.HasObject(person3.ID),
	)

	article6 := object(
		c2sRootIRI.AddPath("objects/article-6"),
		ap.HasType(vocab.ArticleType),
		ap.HasTo(person1.ID),
		ap.HasContent("lorem ipsum dolor sic amet"),
		ap.HasAudience(vocab.PublicNS),
	)

	create5ID := c2sRootIRI.AddPath("/activities/create-5")
	create5 := create(ap.HasActor(person3), ap.HasObject(article6))

	undo6ID := c2sRootIRI.AddPath("/activities/undo-6")
	undo6 := undo(ap.HasActor(person1), ap.HasObject(block4ID), ap.HasCC(person3.ID))

	article7 := object(
		c2sRootIRI.AddPath("objects/article-7"),
		ap.HasType(vocab.ArticleType),
		ap.HasTo(person1.ID),
		ap.HasContent("lorem ipsum dolor sic amet"),
		ap.HasAudience(vocab.PublicNS),
	)
	// NOTE(marius): this somehow gets wounded back
	create8ID := c2sRootIRI.AddPath("/activities/create-7")
	create8 := create(ap.HasActor(person3), ap.HasObject(article7))

	toRun := []tests.RunnableTest{
		tests.TestSuite{
			Name: "control checks",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "person1 exists",
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
					Name: "person1 blocked collection is not accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(blocked.IRI(person1.ID)),
					Res: tests.Response().
						HasCode(http.StatusNotFound).
						HasErrors(errFedBOXNotFound(blocked.IRI(person1.ID))),
				},
				tests.HTTPTest{
					Name: "person1 blocked collection is accessible if authenticated",
					Req: tests.Request().
						Bearer(tokenP1.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(blocked.IRI(person1.ID)),
					Res: tests.Response().
						HasCode(http.StatusOK),
				},
				tests.HTTPTest{
					Name: "person3 exists",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(person3.ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(person3.ID),
							tests.IsType(person3.Type),
							tests.HasName(person3.Name),
							tests.HasPreferredUsername(person3.PreferredUsername),
							tests.HasSummary(person3.Summary),
							tests.HasContent(person3.Content),
							tests.WasPublished(person3.Published),
							tests.HasAudience(person3.Audience),
						),
				},
				tests.HTTPTest{
					Name: "person3 blocked collection is not accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(blocked.IRI(person3.ID)),
					Res: tests.Response().
						HasCode(http.StatusNotFound).
						HasErrors(errFedBOXNotFound(blocked.IRI(person3.ID))),
				},
				tests.HTTPTest{
					Name: "person3 blocked collection is accessible if authenticated",
					Req: tests.Request().
						Bearer(tokenP3.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(blocked.IRI(person3.ID)),
					Res: tests.Response().
						HasCode(http.StatusOK),
				},
				tests.HTTPTest{
					Name: "person1 is accessible as person3",
					Req: tests.Request().
						Bearer(tokenP3.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(person1.ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(person1.ID),
							tests.HasPreferredUsername(person1.PreferredUsername),
							tests.HasName(person1.Name),
							tests.HasAudience(person1.Audience),
							tests.WasPublished(person1.Published),
						),
				},
				tests.HTTPTest{
					Name: "person1 Outbox is accessible as person3",
					Req: tests.Request().
						Bearer(tokenP3.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Outbox.IRI(person1), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(0),
						),
				},
				tests.HTTPTest{
					Name: "person1 Inbox is accessible as person3",
					Req: tests.Request().
						Bearer(tokenP3.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Inbox.IRI(person1)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Inbox.IRI(person1), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(0),
						),
				},
			},
		},
		tests.TestSuite{
			Name: "Block",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Block person3",
					Req: tests.Request().
						Bearer(tokenP1.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1)).
						BodyItem(block4),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						ItemMatch(
							tests.HasID(block4ID),
							tests.IsType(block4.Type),
							tests.HasActor(person1.ID),
							tests.HasObject(person3.ID),
							tests.WasPublished(time.Now().Round(0)),
						),
				},
				tests.HTTPTest{
					Name: "Block is not accessible in Outbox without authorization",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Outbox.IRI(person1), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(1),
							tests.DoesNotHaveItem(block4ID),
						),
				},
				tests.HTTPTest{
					Name: "Block is in Outbox when authorized",
					Req: tests.Request().
						Bearer(tokenP1.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Outbox.IRI(person1), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(1),
							tests.HasItem(block4ID),
						),
				},
				tests.HTTPTest{
					Name: "Block is not in person3 Inbox",
					Req: tests.Request().
						Bearer(tokenP3.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Inbox.IRI(person3.ID)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Inbox.IRI(person3.ID), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(0),
							tests.DoesNotHaveItem(block4ID),
						),
				},
				tests.HTTPTest{
					Name: "Block is accessible as person1",
					Req: tests.Request().
						Bearer(tokenP1.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(block4ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(block4ID),
							tests.IsType(block4.Type),
							tests.HasActor(person1.ID),
							tests.HasObject(person3.ID),
							tests.WasPublished(time.Now().Round(0)),
						),
				},
				tests.HTTPTest{
					Name: "Block is not accessible without authorization",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(block4ID),
					Res: tests.Response().
						HasCode(http.StatusNotFound).
						HasErrors(errFedBOXNotFound(block4ID)),
				},
				tests.HTTPTest{
					Name: "Block is not accessible as person3",
					Req: tests.Request().
						Bearer(tokenP3.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(block4ID),
					Res: tests.Response().
						HasCode(http.StatusNotFound).
						HasErrors(errFedBOXNotFound(block4ID)),
				},
			},
		},
		tests.TestSuite{
			Name: "person1 no longer accessible to person3",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "person1 is not accessible as person3",
					Req: tests.Request().
						Bearer(tokenP3.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(person1.ID),
					Res: tests.Response().
						HasCode(http.StatusNotFound).
						HasErrors(errFedBOXNotFound(person1.ID)),
				},
				tests.HTTPTest{
					Name: "person1 Outbox is not accessible as person3",
					Req: tests.Request().
						Bearer(tokenP3.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1)),
					Res: tests.Response().
						HasCode(http.StatusNotFound).
						HasErrors(errFedBOXNotFound(vocab.Outbox.IRI(person1))),
				},
				tests.HTTPTest{
					Name: "person1 Inbox is not accessible as person3",
					Req: tests.Request().
						Bearer(tokenP3.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Inbox.IRI(person1)),
					Res: tests.Response().
						HasCode(http.StatusNotFound).
						HasErrors(errFedBOXNotFound(vocab.Inbox.IRI(person1))),
				},
			},
		},
		tests.TestSuite{
			Name: "no person1 activities reach person3",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					// NOTE(marius): if we want that a blocked actor is no longer accessible to the actor
					// that operated the block, the client needs to have access to the blocked collection.
					Name: "person3 is accessible as person1",
					Req: tests.Request().
						Bearer(tokenP1.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(person3.ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(person3.ID),
							tests.HasPreferredUsername(person3.PreferredUsername),
							tests.HasContent(person3.Content),
							tests.HasAudience(person3.Audience),
							tests.WasPublished(person3.Published),
						),
				},
				tests.HTTPTest{
					Name: "Create with person1 in CC",
					Req: tests.Request().
						Bearer(tokenP3.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person3)).
						BodyItem(create5),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						HasLocation(create5ID).
						ItemMatch(
							tests.HasID(article6.ID),
							tests.IsType(article6.Type),
							tests.HasTo(article6.To),
							tests.HasCC(article6.CC),
							tests.HasContent(article6.Content),
							tests.HasAudience(vocab.PublicNS),
							tests.WasPublished(time.Now().Round(0)),
						),
				},
				tests.HTTPTest{
					Name: "Create is in person3's Outbox",
					Req: tests.Request().
						Bearer(tokenP3.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person3)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Outbox.IRI(person3), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(1),
							tests.HasItem(create5ID),
						),
				},
				tests.HTTPTest{
					Name: "Create is not in person1's Inbox",
					Req: tests.Request().
						Bearer(tokenP1.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Inbox.IRI(person1)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Inbox.IRI(person1), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(0),
							tests.DoesNotHaveItem(create5ID),
						),
				},
				tests.HTTPTest{
					Name: "Article is publicly accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(article6.ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(article6.ID),
							tests.IsType(article6.Type),
							tests.HasContent(article6.Content),
						),
				},
				tests.HTTPTest{
					Name: "Article is accessible as person1",
					Req: tests.Request().
						Bearer(tokenP1.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(article6.ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(article6.ID),
							tests.IsType(article6.Type),
							tests.HasContent(article6.Content),
						),
				},
			},
		},

		// NOTE(marius): keeping the Undo tests alongside the activity they operate on
		//  to match how we structured the code in the go-ap/processing module
		tests.TestSuite{
			Name: "Undo Block",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Undo Block",
					Req: tests.Request().
						IRI(vocab.Outbox.IRI(person1)).
						Post().
						ContentType(client.ContentTypeJsonLD).
						Signer(tokenP1.Sign).
						BodyItem(undo6),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						ItemMatch(
							tests.HasID(undo6ID),
							tests.IsType(undo6.Type),
							tests.HasCC(undo6.CC),
							tests.HasContent(undo6.Content),
							tests.WasPublished(time.Now()),
						),
				},
				tests.HTTPTest{
					Name: "Undo is in person1's outbox and the Block isn't",
					Req: tests.Request().
						Signer(tokenP1.Sign).
						ContentType(client.ContentTypeJsonLD).
						IRI(vocab.Outbox.IRI(person1)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Outbox.IRI(person1), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(1),
							tests.HasItem(undo6ID),
							tests.DoesNotHaveItem(block4ID),
						),
				},
				tests.HTTPTest{
					Name: "Block is no longer accessible",
					Req: tests.Request().
						ContentType(client.ContentTypeJson).
						IRI(block4ID),
					Res: tests.Response().
						HasCode(http.StatusNotFound).
						HasErrors(errFedBOXNotFound(block4ID)),
				},
				tests.TestSuite{
					Name: "Block side-effects no longer apply",
					Tests: []tests.RunnableTest{
						tests.HTTPTest{
							Name: "person1 is again accessible as person3",
							Req: tests.Request().
								Bearer(tokenP3.AccessToken).
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
							Name: "person1 Outbox is again accessible as person3",
							Req: tests.Request().
								Bearer(tokenP3.AccessToken).
								Accept(client.ContentTypeJsonActivity).
								IRI(vocab.Outbox.IRI(person1)),
							Res: tests.Response().
								HasCode(http.StatusOK).
								ItemMatch(
									tests.HasID(filterIRI(vocab.Outbox.IRI(person1), filters.WithMaxCount(filters.MaxItems))),
									tests.IsType(vocab.OrderedCollectionPageType),
									tests.HasTotalItems(1),
									tests.HasItem(undo6ID),
								),
						},
						tests.HTTPTest{
							Name: "person1 Inbox is again accessible as person3",
							Req: tests.Request().
								Bearer(tokenP3.AccessToken).
								Accept(client.ContentTypeJsonActivity).
								IRI(vocab.Inbox.IRI(person1)),
							Res: tests.Response().
								HasCode(http.StatusOK).
								ItemMatch(
									tests.HasID(filterIRI(vocab.Inbox.IRI(person1), filters.WithMaxCount(filters.MaxItems))),
									tests.IsType(vocab.OrderedCollectionPageType),
									tests.HasTotalItems(0),
								),
						},
						tests.TestSuite{
							Name: "person1's activities reach person3 again",
							Tests: []tests.RunnableTest{
								tests.HTTPTest{
									Name: "Create with person1 in CC",
									Req: tests.Request().
										Bearer(tokenP3.AccessToken).
										Accept(client.ContentTypeJsonActivity).
										IRI(vocab.Outbox.IRI(person3)).
										BodyItem(create8),
									Res: tests.Response().
										HasCode(http.StatusCreated).
										HasLocation(create8ID).
										ItemMatch(
											tests.HasID(article7.ID),
											tests.IsType(article7.Type),
											tests.HasCC(article7.CC),
											tests.HasContent(article7.Content),
											tests.HasAudience(vocab.PublicNS),
											tests.WasPublished(time.Now().Round(0)),
										),
								},
								tests.HTTPTest{
									Name: "Create is in person3's Outbox",
									Req: tests.Request().
										Bearer(tokenP3.AccessToken).
										Accept(client.ContentTypeJsonActivity).
										IRI(vocab.Outbox.IRI(person3)),
									Res: tests.Response().
										HasCode(http.StatusOK).
										ItemMatch(
											tests.HasID(filterIRI(vocab.Outbox.IRI(person3), filters.WithMaxCount(filters.MaxItems))),
											tests.IsType(vocab.OrderedCollectionPageType),
											tests.HasTotalItems(2),
											tests.HasItem(create5ID),
											tests.HasItem(create8ID),
										),
								},
								tests.HTTPTest{
									Name: "Create is also in person1's Inbox",
									Req: tests.Request().
										Bearer(tokenP1.AccessToken).
										Accept(client.ContentTypeJsonActivity).
										IRI(vocab.Inbox.IRI(person1)),
									Res: tests.Response().
										HasCode(http.StatusOK).
										ItemMatch(
											tests.HasID(filterIRI(vocab.Inbox.IRI(person1), filters.WithMaxCount(filters.MaxItems))),
											tests.IsType(vocab.OrderedCollectionPageType),
											tests.HasTotalItems(1),
											tests.HasItem(create8ID),
										),
								},
								tests.HTTPTest{
									Name: "Article is accessible as person1",
									Req: tests.Request().
										Bearer(tokenP1.AccessToken).
										Accept(client.ContentTypeJsonActivity).
										IRI(article7.ID),
									Res: tests.Response().
										HasCode(http.StatusOK).
										ItemMatch(
											tests.HasID(article7.ID),
											tests.IsType(article7.Type),
											tests.HasContent(article7.Content),
										),
								},
							},
						},
					},
				},
			},
		},
	}

	for _, test := range toRun {
		t.Run(test.Label(), test.Fn(t.Context(), cont))
	}
}

// TODO(marius): IgnoreRequests -> ignored's activities don't get to ignorer's inbox
func Test_C2S_IgnoreRequests(t *testing.T) {
	tokenP1 := new(c2s.BearerSigner)
	tokenP3 := new(c2s.BearerSigner)
	rootExec := c.ExecAs(c2sRootIRI, ed2559Key)

	conf := fedbox.C2SConfig(
		fedBOXImageName,
		person1, person3, ed2559Key,
		rootExec.ExtractOAuth2Bearer(person1.ID, tokenP1),
		rootExec.ExtractOAuth2Bearer(person3.ID, tokenP3),
	)

	cont, err := fedbox.StartContainers(t.Context(), t, conf)
	if err != nil {
		t.Fatalf("Unable to start test containers: %v", err)
	}

	t.Cleanup(func() {
		cont.Cleanup(t)
	})

	ignored := vocab.CollectionPath("ignored")

	ignore4ID := c2sRootIRI.AddPath("/activities/ignore-4")
	ignore4 := ignore(
		ap.HasActor(person3.ID),
		ap.HasObject(person1.ID),
	)

	undo5ID := c2sRootIRI.AddPath("/activities/undo-5")
	undo5 := undo(ap.HasActor(person3), ap.HasObject(ignore4ID), ap.HasCC(person1.ID))

	toRun := []tests.RunnableTest{
		tests.TestSuite{
			Name: "control checks",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "person1 exists",
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
					Name: "person1 ignored collection is not accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(ignored.IRI(person1.ID)),
					Res: tests.Response().
						HasCode(http.StatusNotFound).
						HasErrors(errFedBOXNotFound(ignored.IRI(person1.ID))),
				},
				tests.HTTPTest{
					Name: "person1 ignored collection is accessible if authenticated",
					Req: tests.Request().
						Bearer(tokenP1.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(ignored.IRI(person1.ID)),
					Res: tests.Response().
						HasCode(http.StatusOK),
				},
				tests.HTTPTest{
					Name: "person3 exists",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(person3.ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(person3.ID),
							tests.IsType(person3.Type),
							tests.HasName(person3.Name),
							tests.HasPreferredUsername(person3.PreferredUsername),
							tests.HasSummary(person3.Summary),
							tests.HasContent(person3.Content),
							tests.WasPublished(person3.Published),
							tests.HasAudience(person3.Audience),
						),
				},
				tests.HTTPTest{
					Name: "person3 ignored collection is not accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(ignored.IRI(person3.ID)),
					Res: tests.Response().
						HasCode(http.StatusNotFound).
						HasErrors(errFedBOXNotFound(ignored.IRI(person3.ID))),
				},
				tests.HTTPTest{
					Name: "person3 ignored collection is accessible if authenticated",
					Req: tests.Request().
						Bearer(tokenP3.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(ignored.IRI(person3.ID)),
					Res: tests.Response().
						HasCode(http.StatusOK),
				},
				tests.HTTPTest{
					Name: "person1 is accessible as person3",
					Req: tests.Request().
						Bearer(tokenP3.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(person1.ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(person1.ID),
							tests.HasPreferredUsername(person1.PreferredUsername),
							tests.HasName(person1.Name),
							tests.HasAudience(person3.Audience),
							tests.WasPublished(person3.Published),
						),
				},
				tests.HTTPTest{
					Name: "person3 is accessible as person1",
					Req: tests.Request().
						Bearer(tokenP1.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(person3.ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(person3.ID),
							tests.HasPreferredUsername(person3.PreferredUsername),
							tests.HasName(person3.Name),
							tests.HasAudience(person3.Audience),
							tests.WasPublished(person3.Published),
						),
				},
			},
		},
		tests.TestSuite{
			Name: "Ignore",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Ignore person1",
					Req: tests.Request().
						Bearer(tokenP3.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person3)).
						BodyItem(ignore4),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						ItemMatch(
							tests.HasID(ignore4ID),
							tests.IsType(ignore4.Type),
							tests.HasActor(person3.ID),
							tests.HasObject(person1.ID),
							tests.WasPublished(time.Now().Round(0)),
						),
				},
				tests.HTTPTest{
					Name: "Ignore is not accessible in Outbox without authorization",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person3)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Outbox.IRI(person3), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(1),
							tests.DoesNotHaveItem(ignore4ID),
						),
				},
				tests.HTTPTest{
					Name: "Ignore is in Outbox when authorized",
					Req: tests.Request().
						Bearer(tokenP3.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person3)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Outbox.IRI(person3), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(1),
							tests.HasItem(ignore4ID),
						),
				},
				tests.HTTPTest{
					Name: "Ignore is not in person3 Inbox",
					Req: tests.Request().
						Bearer(tokenP1.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Inbox.IRI(person1.ID)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Inbox.IRI(person1.ID), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(0),
							tests.DoesNotHaveItem(ignore4ID),
						),
				},
				tests.HTTPTest{
					Name: "Ignore is accessible as person1",
					Req: tests.Request().
						Bearer(tokenP1.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(ignore4ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(ignore4ID),
							tests.IsType(ignore4.Type),
							tests.HasActor(person3.ID),
							tests.HasObject(person1.ID),
							tests.WasPublished(time.Now().Round(0)),
						),
				},
				tests.HTTPTest{
					Name: "Ignore is not accessible without authorization",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(ignore4ID),
					Res: tests.Response().
						HasCode(http.StatusNotFound).
						HasErrors(errFedBOXNotFound(ignore4ID)),
				},
				tests.HTTPTest{
					Name: "Ignore is accessible also as person1",
					Req: tests.Request().
						Bearer(tokenP1.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(ignore4ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(ignore4ID),
							tests.IsType(ignore4.Type),
							tests.HasActor(person3.ID),
							tests.HasObject(person1.ID),
							tests.WasPublished(time.Now().Round(0)),
						),
				},
			},
		},

		// NOTE(marius): keeping the Undo tests alongside the activity they operate on
		//  to match how we structured the code in the go-ap/processing module
		tests.TestSuite{
			Name: "Undo Ignore",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Undo Ignore",
					Req: tests.Request().
						IRI(vocab.Outbox.IRI(person3)).
						Post().
						ContentType(client.ContentTypeJsonLD).
						Signer(tokenP3.Sign).
						BodyItem(undo5),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						ItemMatch(
							tests.HasID(undo5ID),
							tests.IsType(undo5.Type),
							tests.HasCC(undo5.CC),
							tests.HasContent(undo5.Content),
							tests.WasPublished(time.Now()),
						),
				},
				tests.HTTPTest{
					Name: "Undo is in person3's outbox and the Ignore isn't",
					Req: tests.Request().
						Signer(tokenP3.Sign).
						ContentType(client.ContentTypeJsonLD).
						IRI(vocab.Outbox.IRI(person3)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Outbox.IRI(person3), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(1),
							tests.HasItem(undo5ID),
							tests.DoesNotHaveItem(ignore4ID),
						),
				},
				tests.HTTPTest{
					Name: "Ignore is no longer accessible",
					Req: tests.Request().
						ContentType(client.ContentTypeJson).
						IRI(ignore4ID),
					Res: tests.Response().
						HasCode(http.StatusNotFound).
						HasErrors(errFedBOXNotFound(ignore4ID)),
				},
				tests.TestSuite{
					Name: "Ignore side-effects no longer apply",
					Tests: []tests.RunnableTest{
						tests.HTTPTest{
							Name: "person3 is again accessible as person1",
							Req: tests.Request().
								Bearer(tokenP1.AccessToken).
								Accept(client.ContentTypeJsonActivity).
								IRI(person3.ID),
							Res: tests.Response().
								HasCode(http.StatusOK).
								ItemMatch(
									tests.HasID(person3.ID),
									tests.IsType(person3.Type),
									tests.HasName(person3.Name),
									tests.HasPreferredUsername(person3.PreferredUsername),
									tests.HasSummary(person3.Summary),
									tests.HasContent(person3.Content),
									tests.WasPublished(person3.Published),
									tests.HasAudience(person3.Audience),
								),
						},
						tests.HTTPTest{
							Name: "person3 Outbox is again accessible as person1",
							Req: tests.Request().
								Bearer(tokenP1.AccessToken).
								Accept(client.ContentTypeJsonActivity).
								IRI(vocab.Outbox.IRI(person3)),
							Res: tests.Response().
								HasCode(http.StatusOK).
								ItemMatch(
									tests.HasID(filterIRI(vocab.Outbox.IRI(person3), filters.WithMaxCount(filters.MaxItems))),
									tests.IsType(vocab.OrderedCollectionPageType),
									tests.HasTotalItems(1),
									tests.HasItem(undo5ID),
								),
						},
					},
				},
			},
		},
	}

	for _, test := range toRun {
		t.Run(test.Label(), test.Fn(t.Context(), cont))
	}
}

func Test_C2S_QuestionRequests(t *testing.T) {
	tokenP1 := new(c2s.BearerSigner)
	rootExec := c.ExecAs(c2sRootIRI, ed2559Key)

	conf := fedbox.C2SConfig(
		fedBOXImageName, person1, ed2559Key,
		rootExec.ExtractOAuth2Bearer(person1.ID, tokenP1),
	)

	cont, err := fedbox.StartContainers(t.Context(), t, conf)
	if err != nil {
		t.Fatalf("Unable to start test containers: %v", err)
	}

	t.Cleanup(func() {
		cont.Cleanup(t)
	})

	question3ID := c2sRootIRI.AddPath("/activities/question-3")
	question3 := question(
		ap.HasName("Some question"),
		ap.HasActor(person1.ID),
	)

	option := func(initFns ...ap.InitFn) vocab.Item {
		initFns = append(initFns, ap.HasType(vocab.NoteType))
		return object(initFns...)
	}

	option5ID := c2sRootIRI.AddPath("/activities/note-5")
	option6ID := c2sRootIRI.AddPath("/activities/note-6")

	question4ID := c2sRootIRI.AddPath("/activities/question-4")
	question4 := question(
		ap.HasName("A second question, any answer"),
		ap.HasActor(person1.ID),
		ap.AnyOf(
			option(option5ID, ap.HasName("Option 1")),
			option(option6ID, ap.HasName("Option 2")),
		),
	)

	option7ID := c2sRootIRI.AddPath("/activities/note-7")
	option8ID := c2sRootIRI.AddPath("/activities/note-8")
	option9ID := c2sRootIRI.AddPath("/activities/note-9")
	question5ID := c2sRootIRI.AddPath("/activities/question-5")
	question5 := question(
		ap.HasName("A third question, only one answer"),
		ap.HasActor(person1.ID),
		ap.OneOf(
			option(option7ID, ap.HasName("Option 1")),
			option(option8ID, ap.HasName("Option 2")),
			option(option9ID, ap.HasName("Option 3")),
		),
	)

	toRun := []tests.RunnableTest{
		tests.TestSuite{
			Name: "Question w/o answers",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Question 3",
					Req: tests.Request().
						Bearer(tokenP1.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1)).
						BodyItem(question3),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						ItemMatch(
							tests.HasID(question3ID),
							tests.IsType(question3.Type),
							tests.HasContent(question3.Content),
							tests.HasActor(question3.Actor),
							tests.HasAnyOf(question3.AnyOf),
							tests.HasOneOf(question3.OneOf),
							tests.WasPublished(time.Now()),
						),
				},
				tests.HTTPTest{
					Name: "Question3 is in person1's Outbox",
					Req: tests.Request().
						Bearer(tokenP1.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1.ID)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Outbox.IRI(person1.ID), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(1),
							tests.HasItem(question3ID),
						),
				},
				tests.HTTPTest{
					Name: "Question3 is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(question3ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(question3ID),
							tests.IsType(question3.Type),
							tests.HasActor(question3.Actor),
							tests.HasContent(question3.Content),
							tests.WasPublished(time.Now().Round(0)),
							tests.HasAnyOf(question3.AnyOf),
							tests.HasOneOf(question3.OneOf),
						),
				},
			},
		},

		tests.TestSuite{
			Name: "Question with anyOf answers",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Question 4",
					Req: tests.Request().
						Bearer(tokenP1.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1)).
						BodyItem(question4),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						ItemMatch(
							tests.HasID(question4ID),
							tests.IsType(question4.Type),
							tests.HasContent(question4.Content),
							tests.HasActor(question4.Actor),
							tests.HasAnyOf(question4.AnyOf),
							tests.HasOneOf(question4.OneOf),
							tests.WasPublished(time.Now()),
						),
				},
				tests.HTTPTest{
					Name: "Question4 is in person1's Outbox",
					Req: tests.Request().
						Bearer(tokenP1.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1.ID)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Outbox.IRI(person1.ID), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(2),
							tests.HasItem(question3ID),
							tests.HasItem(question4ID),
						),
				},
				tests.HTTPTest{
					Name: "Question4 is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(question4ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(question4ID),
							tests.IsType(question4.Type),
							tests.HasActor(question4.Actor),
							tests.HasContent(question4.Content),
							tests.WasPublished(time.Now().Round(0)),
							tests.HasAnyOf(question4.AnyOf),
							tests.HasOneOf(question4.OneOf),
						),
				},
			},
		},

		tests.TestSuite{
			Name: "Question with oneOf answers",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Question 5",
					Req: tests.Request().
						Bearer(tokenP1.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1)).
						BodyItem(question5),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						ItemMatch(
							tests.HasID(question5ID),
							tests.IsType(question5.Type),
							tests.HasContent(question5.Content),
							tests.HasActor(question5.Actor),
							tests.HasAnyOf(question5.AnyOf),
							tests.HasOneOf(question5.OneOf),
							tests.WasPublished(time.Now()),
						),
				},
				tests.HTTPTest{
					Name: "Question5 is in person1's Outbox",
					Req: tests.Request().
						Bearer(tokenP1.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1.ID)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Outbox.IRI(person1.ID), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(3),
							tests.HasItem(question3ID),
							tests.HasItem(question4ID),
							tests.HasItem(question5ID),
						),
				},
				tests.HTTPTest{
					Name: "Question5 is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(question5ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(question5ID),
							tests.IsType(question5.Type),
							tests.HasActor(question5.Actor),
							tests.HasContent(question5.Content),
							tests.WasPublished(time.Now().Round(0)),
							tests.HasAnyOf(question5.AnyOf),
							tests.HasOneOf(question5.OneOf),
						),
				},
			},
		},
	}

	for _, test := range toRun {
		t.Run(test.Label(), test.Fn(t.Context(), cont))
	}
}

// TODO(marius): AddRequests -> collection gets updated with item
func Test_C2S_AddRequests(t *testing.T) {
	tokenP1 := new(c2s.BearerSigner)
	rootExec := c.ExecAs(c2sRootIRI, ed2559Key)

	target := collection(c2sRootIRI.AddPath("objects/target-collection"))
	note := object(
		c2sRootIRI.AddPath("objects/note"),
		ap.HasType("Note"),
		ap.HasContent("Lorem ipsum dolor sic amet."),
	)

	conf := fedbox.C2SConfig(
		fedBOXImageName, person1, note, target,
		ed2559Key,
		rootExec.ExtractOAuth2Bearer(person1.ID, tokenP1),
	)

	cont, err := fedbox.StartContainers(t.Context(), t, conf)
	if err != nil {
		t.Fatalf("Unable to start test containers: %v", err)
	}

	t.Cleanup(func() {
		cont.Cleanup(t)
	})

	add5ID := c2sRootIRI.AddPath("activities/add-5")
	add5 := add(ap.HasActor(person1), ap.HasObject(note.ID), ap.HasTarget(target.ID))

	toRun := []tests.RunnableTest{
		tests.TestSuite{
			Name: "Add to collection",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Add5",
					Req: tests.Request().
						Bearer(tokenP1.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1)).
						BodyItem(add5),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						ItemMatch(
							tests.HasID(add5ID),
							tests.IsType(add5.Type),
							tests.HasActor(add5.Actor),
							tests.HasObject(add5.Object),
							tests.HasTarget(add5.Target),
							tests.WasPublished(time.Now()),
						),
				},
				tests.HTTPTest{
					Name: "Add5 is in person1's Outbox",
					Req: tests.Request().
						Bearer(tokenP1.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1.ID)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Outbox.IRI(person1.ID), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(1),
							tests.HasItem(add5ID),
						),
				},
				tests.HTTPTest{
					Name: "Add5 is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(add5ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(add5ID),
							tests.IsType(add5.Type),
							tests.HasActor(add5.Actor),
							tests.HasObject(add5.Object),
							tests.HasTarget(add5.Target),
							tests.WasPublished(time.Now().Round(0)),
						),
				},
			},
		},
		tests.TestSuite{
			Name:  "Add side-effects",
			Tests: []tests.RunnableTest{},
		},
	}
	for _, test := range toRun {
		t.Run(test.Label(), test.Fn(t.Context(), cont))
	}
}

// TODO(marius): RemoveRequests -> collection loses item
func Test_C2S_RemoveRequests(t *testing.T) {
	t.Skipf("Remove tests are not yet ready")
	tokenP1 := new(c2s.BearerSigner)
	rootExec := c.ExecAs(c2sRootIRI, ed2559Key)

	target := collection(c2sRootIRI.AddPath("objects/target-collection"))
	note := object(
		c2sRootIRI.AddPath("objects/note"),
		ap.HasType("Note"),
		ap.HasContent("Lorem ipsum dolor sic amet."),
	)
	conf := fedbox.C2SConfig(
		fedBOXImageName, person1, target, note, ed2559Key,
		rootExec.ExtractOAuth2Bearer(person1.ID, tokenP1),
	)

	cont, err := fedbox.StartContainers(t.Context(), t, conf)
	if err != nil {
		t.Fatalf("Unable to start test containers: %v", err)
	}

	t.Cleanup(func() {
		cont.Cleanup(t)
	})

	add5ID := c2sRootIRI.AddPath("activities/add-5")
	add5 := add(ap.HasActor(person1), ap.HasObject(note.ID), ap.HasTarget(target.ID))

	remove6ID := c2sRootIRI.AddPath("activities/remove-6")
	remove6 := remove(ap.HasActor(person1), ap.HasObject(note.ID), ap.HasTarget(target.ID))

	toRun := []tests.RunnableTest{
		tests.TestSuite{
			Name: "setup",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Add to target collection",
					Req: tests.Request().
						Bearer(tokenP1.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1)).
						BodyItem(add5),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						ItemMatch(
							tests.HasID(add5ID),
							tests.IsType(add5.Type),
							tests.HasActor(add5.Actor),
							tests.HasObject(add5.Object),
							tests.HasTarget(add5.Target),
							tests.WasPublished(time.Now()),
						),
				},
			},
		},

		tests.TestSuite{
			Name: "Remove from collection",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Remove6",
					Req: tests.Request().
						Bearer(tokenP1.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1)).
						BodyItem(remove6),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						ItemMatch(
							tests.HasID(remove6ID),
							tests.IsType(remove6.Type),
							tests.HasActor(remove6.Actor),
							tests.HasObject(remove6.Object),
							tests.HasTarget(remove6.Target),
							tests.WasPublished(time.Now()),
						),
				},
				tests.HTTPTest{
					Name: "Remove6 is in person1's Outbox",
					Req: tests.Request().
						Bearer(tokenP1.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1.ID)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Outbox.IRI(person1.ID), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(2),
							tests.HasItem(remove6ID),
						),
				},
				tests.HTTPTest{
					Name: "Remove6 is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(remove6ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(remove6ID),
							tests.IsType(remove6.Type),
							tests.HasActor(remove6.Actor),
							tests.HasObject(remove6.Object),
							tests.HasTarget(remove6.Target),
							tests.WasPublished(time.Now().Round(0)),
						),
				},
			},
		},
	}
	for _, test := range toRun {
		t.Run(test.Label(), test.Fn(t.Context(), cont))
	}
}

// TODO(marius): MoveRequests -> old collection loses item, new collection gets updated with item
func Test_C2S_MoveRequests(t *testing.T) {
	tokenP1 := new(c2s.BearerSigner)
	rootExec := c.ExecAs(c2sRootIRI, ed2559Key)

	target := collection(c2sRootIRI.AddPath("objects/target-collection"))
	origin := collection(c2sRootIRI.AddPath("objects/origin-collection"))
	note := object(
		c2sRootIRI.AddPath("objects/note"),
		ap.HasType("Note"),
		ap.HasContent("Lorem ipsum dolor sic amet."),
	)
	conf := fedbox.C2SConfig(
		fedBOXImageName, person1, target, origin, note,
		ed2559Key,
		rootExec.ExtractOAuth2Bearer(person1.ID, tokenP1),
	)

	cont, err := fedbox.StartContainers(t.Context(), t, conf)
	if err != nil {
		t.Fatalf("Unable to start test containers: %v", err)
	}

	t.Cleanup(func() {
		cont.Cleanup(t)
	})

	add6ID := c2sRootIRI.AddPath("activities/add-6")
	add6 := add(ap.HasActor(person1), ap.HasObject(note.ID), ap.HasTarget(origin.ID))

	move7ID := c2sRootIRI.AddPath("activities/move-7")
	move7 := move(ap.HasActor(person1), ap.HasObject(note.ID), ap.HasOrigin(origin.ID), ap.HasTarget(target.ID))

	toRun := []tests.RunnableTest{
		tests.TestSuite{
			Name: "setup",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Add to origin collection",
					Req: tests.Request().
						Bearer(tokenP1.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1)).
						BodyItem(add6),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						ItemMatch(
							tests.HasID(add6ID),
							tests.IsType(add6.Type),
							tests.HasActor(add6.Actor),
							tests.HasObject(add6.Object),
							tests.HasTarget(add6.Target),
							tests.WasPublished(time.Now()),
						),
				},
			},
		},

		tests.TestSuite{
			Name: "Move from collection",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Move",
					Req: tests.Request().
						Bearer(tokenP1.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1)).
						BodyItem(move7),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						ItemMatch(
							tests.HasID(move7ID),
							tests.IsType(move7.Type),
							tests.HasActor(move7.Actor),
							tests.HasObject(move7.Object),
							tests.HasTarget(move7.Target),
							tests.HasOrigin(move7.Origin),
							tests.WasPublished(time.Now()),
						),
				},
				tests.HTTPTest{
					Name: "Move is in person1's Outbox",
					Req: tests.Request().
						Bearer(tokenP1.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1.ID)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Outbox.IRI(person1.ID), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(2),
							tests.HasItem(move7ID),
						),
				},
				tests.HTTPTest{
					Name: "Move is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(move7ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(move7ID),
							tests.IsType(move7.Type),
							tests.HasActor(move7.Actor),
							tests.HasObject(move7.Object),
							tests.HasTarget(move7.Target),
							tests.HasOrigin(move7.Origin),
							tests.WasPublished(time.Now().Round(0)),
						),
				},
			},
		},
	}
	for _, test := range toRun {
		t.Run(test.Label(), test.Fn(t.Context(), cont))
	}
}

func Test_C2S_Requests(t *testing.T) {
	conf := fedbox.C2SConfig(fedBOXImageName, tagAdmin, admin)
	cont, err := fedbox.StartContainers(t.Context(), t, conf)
	if err != nil {
		t.Fatalf("Error: %s", err)
	}

	t.Cleanup(func() {
		cont.Cleanup(t)
	})

	service := root(c2sRootIRI, ap.HasPublicKey(conf.Key.Public()))
	draftSig := s2s.New(s2s.WithActor(service, conf.Key))
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
					tests.HasTo(tagAdmin.To...),
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
					tests.HasTo(vocab.PublicNS),
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
				HasErrors(errors.Unauthorizedf("authorized Actor is invalid")),
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
					errors.NewBadRequest(errors.BadRequestf("Activity is not valid: empty activity id"), "Unable to save activity Flag to %s", vocab.Inbox.IRI(admin)),
					errors.BadRequestf("Activity is not valid: empty activity id"),
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
						Key:  conf.Key,
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
							tests.HasTo(vocab.PublicNS),
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
							tests.HasTo(vocab.PublicNS),
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
							tests.HasTo(vocab.PublicNS),
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
						HasErrors(errFedBOXNotFound(c2sRootIRI.AddPath("activities/flag-7"))),
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

	for _, test := range toRun {
		t.Run(test.Label(), test.Fn(t.Context(), cont))
	}
}
