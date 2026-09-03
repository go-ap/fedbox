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
