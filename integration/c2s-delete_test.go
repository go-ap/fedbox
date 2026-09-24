//go:build c2s || Delete

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
	conf := fedbox.C2SConfig(
		fedbox.WithImageName(imageName),
		fedbox.WithItems(person1, article3),
		fedbox.WithPrivateKey(ed2559Key),
		fedbox.WithCommands(rootExec.ExtractOAuth2Bearer(person1.ID, token)),
		fedbox.Verbose(verbose), fedbox.WithCodeCoverage(coverage),
	)

	cont, err := fedbox.StartContainers(t.Context(), t, conf)
	if err != nil {
		t.Fatalf("Unable to start test containers: %+v", err)
	}

	delete4ID := c2sRootIRI.AddPath("activities/delete-4")
	delete4 := del(
		ap.HasCC(vocab.PublicNS),
		ap.HasActor(person1),
		ap.HasObject(article3.ID),
	)

	note4 := object(
		c2sRootIRI.AddPath("objects/note-4"),
		ap.HasAudience(vocab.PublicNS),
		ap.HasType(vocab.NoteType),
		ap.HasContent("First of two"),
	)
	note5 := object(
		c2sRootIRI.AddPath("objects/note-5"),
		ap.HasAudience(vocab.PublicNS),
		ap.HasType(vocab.NoteType),
		ap.HasContent("Second of two"),
	)
	create5ID := c2sRootIRI.AddPath("/activities/create-5")
	create5 := create(ap.HasActor(person1), ap.HasObject(note4, note5))

	delete6ID := c2sRootIRI.AddPath("activities/delete-6")
	delete6 := del(
		ap.HasCC(vocab.PublicNS),
		ap.HasActor(person1),
		ap.HasObject(note4, note5),
	)

	delete7ID := c2sRootIRI.AddPath("activities/delete-7")
	delete7 := del(
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
			Name: "Delete multiple notes",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Create notes",
					Req: tests.Request().
						Bearer(token.AccessToken).
						IRI(vocab.Outbox.IRI(person1)).
						BodyItem(create5),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						HasLocation(create5ID).
						ItemMatch(
							tests.HasItem(note4),
							tests.HasItem(note5),
						),
				},
				tests.HTTPTest{
					Name: "note10 is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(note4.ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(note4.ID),
							tests.IsType(note4.Type),
							tests.HasContent(note4.Content),
						),
				},
				tests.HTTPTest{
					Name: "note11 is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(note5.ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(note5.ID),
							tests.IsType(note5.Type),
							tests.HasContent(note5.Content),
						),
				},
				tests.TestSuite{
					Name: "Delete notes",
					Tests: []tests.RunnableTest{
						tests.HTTPTest{
							Name: "Delete note10, note11",
							Req: tests.Request().
								IRI(vocab.Outbox.IRI(person1)).
								Post().
								ContentType(client.ContentTypeJsonLD).
								Signer(token.Sign).
								BodyItem(delete6),
							Res: tests.Response().
								HasCode(http.StatusGone).
								ItemMatch(
									tests.HasID(delete6ID),
									tests.IsType(delete6.Type),
									tests.HasCC(delete6.CC...),
									tests.HasActor(delete6.Actor),
									tests.HasObject(delete6.Object),
									tests.WasPublished(time.Now()),
								),
						},
						tests.HTTPTest{
							Name: "Delete is in person1's outbox",
							Req: tests.Request().
								Signer(token.Sign).
								ContentType(client.ContentTypeJsonLD).
								IRI(vocab.Outbox.IRI(person1)),
							Res: tests.Response().
								HasCode(http.StatusOK).
								ItemMatch(
									tests.HasID(filterIRI(vocab.Outbox.IRI(person1), filters.WithMaxCount(filters.MaxItems))),
									tests.IsType(vocab.OrderedCollectionPageType),
									tests.HasTotalItems(3),
									tests.HasItem(delete6ID),
									tests.HasItem(create5ID),
								),
						},
						tests.HTTPTest{
							Name: "note10 is now a tombstone",
							Req: tests.Request().
								ContentType(client.ContentTypeJson).
								IRI(note4.ID),
							Res: tests.Response().
								HasCode(http.StatusGone).
								ItemMatch(
									tests.HasID(note4.ID),
									tests.IsType(vocab.TombstoneType),
									tests.WasDeleted(time.Now().Round(0)),
									tests.HasFormerType(note4.Type),
								),
						},
						tests.HTTPTest{
							Name: "note11 is now a tombstone",
							Req: tests.Request().
								ContentType(client.ContentTypeJson).
								IRI(note5.ID),
							Res: tests.Response().
								HasCode(http.StatusGone).
								ItemMatch(
									tests.HasID(note5.ID),
									tests.IsType(vocab.TombstoneType),
									tests.WasDeleted(time.Now().Round(0)),
									tests.HasFormerType(note5.Type),
								),
						},
					},
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
						BodyItem(delete7),
					Res: tests.Response().
						HasCode(http.StatusGone).
						ItemMatch(
							tests.HasID(delete7ID),
							tests.IsType(delete7.Type),
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
							tests.HasTotalItems(7),
							tests.HasItem(delete4ID),
							tests.HasItem(create5ID),
							tests.HasItem(delete6ID),
							tests.HasItem(delete7ID),
						),
				},
				tests.HTTPTest{
					Name: "Delete is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(delete7ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(delete7ID),
							tests.IsType(delete7.Type),
							tests.HasActor(delete7.Actor),
							tests.HasObject(delete7.Object),
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
