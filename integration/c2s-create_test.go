//go:build c2s && (create || all)

package integration

import (
	"net/http"
	"testing"
	"time"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/client"
	"github.com/go-ap/client/c2s"
	c "github.com/go-ap/fedbox/integration/internal/containers"
	"github.com/go-ap/fedbox/integration/internal/containers/fedbox"
	"github.com/go-ap/fedbox/integration/internal/tests"
	ap "github.com/go-ap/fedbox/integration/internal/vocab"
	"github.com/go-ap/filters"
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
