//go:build c2s && (Follow || all)

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
		t.Fatalf("Unable to start test containers: %+v", err)
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
							tests.HasTotalItems(4),
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
