//go:build c2s && (Delete || all)

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
		t.Fatalf("Unable to start test containers: %+v", err)
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
