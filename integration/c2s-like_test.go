//go:build c2s

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
