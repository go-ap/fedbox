//go:build c2s && (Update || all)

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
		t.Fatalf("Unable to start test containers: %+v", err)
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
