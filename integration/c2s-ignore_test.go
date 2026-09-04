//go:build c2s && (ignore || all)

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
							tests.HasTotalItems(2),
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

		// TODO(marius): what kind of side effects are Ignore activities supposed to have?
	}

	for _, test := range toRun {
		t.Run(test.Label(), test.Fn(t.Context(), cont))
	}
}
