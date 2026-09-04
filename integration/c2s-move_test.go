//go:build c2s && (move || all)

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
				tests.HTTPTest{
					Name: "origin collection contains note",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(origin.ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(origin.ID),
							tests.IsType(vocab.CollectionType),
							tests.HasTotalItems(1),
							tests.HasItem(note.ID),
						),
				},
				tests.HTTPTest{
					Name: "target collection does not contain note",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(target.ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(target.ID),
							tests.IsType(vocab.CollectionType),
							tests.HasTotalItems(0),
							tests.DoesNotHaveItem(note.ID),
						),
				},
				tests.HTTPTest{
					Name: "note is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(note.ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(note.ID),
							tests.IsType(note.Type),
							tests.HasContent(note.Content),
						),
				},
			},
		},

		tests.TestSuite{
			Name: "Move from origin to target",
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

		tests.TestSuite{
			Name: "Move side-effects",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "origin collection does not contain note anymore",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(origin.ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(origin.ID),
							tests.IsType(vocab.CollectionType),
							tests.HasTotalItems(0),
							tests.DoesNotHaveItem(note.ID),
						),
				},
				tests.HTTPTest{
					Name: "target collection contains note",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(target.ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(target.ID),
							tests.IsType(vocab.CollectionType),
							tests.HasTotalItems(1),
							tests.HasItem(note.ID),
						),
				},
				tests.HTTPTest{
					Name: "note is still accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(note.ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(note.ID),
							tests.IsType(note.Type),
							tests.HasContent(note.Content),
						),
				},
			},
		},
	}
	for _, test := range toRun {
		t.Run(test.Label(), test.Fn(t.Context(), cont))
	}
}
