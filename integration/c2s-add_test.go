//go:build c2s && (Add || all)

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

func Test_C2S_AddRequests(t *testing.T) {
	tokenP1 := new(c2s.BearerSigner)
	rootExec := c.ExecAs(c2sRootIRI, ed2559Key)

	target := collection(c2sRootIRI.AddPath("objects/target-collection"))
	note := object(
		c2sRootIRI.AddPath("objects/note"),
		ap.HasAttributedTo(person1.ID),
		ap.HasType("Note"),
		ap.HasCC(vocab.PublicNS),
		ap.HasContent("Lorem ipsum dolor sic amet."),
	)

	conf := fedbox.C2SConfig(
		fedBOXImageName, person1, note, target,
		ed2559Key,
		rootExec.ExtractOAuth2Bearer(person1.ID, tokenP1),
	)

	cont, err := fedbox.StartContainers(t.Context(), t, conf)
	if err != nil {
		t.Fatalf("Unable to start test containers: %+v", err)
	}

	t.Cleanup(func() {
		cont.Cleanup(t)
	})

	add5ID := c2sRootIRI.AddPath("activities/add-5")
	add5 := add(ap.HasActor(person1), ap.HasObject(note.ID), ap.HasTarget(target.ID))

	// TODO(marius): We should add some tests for failure cases:
	//  * try to add an item to a collection that doesn't exist
	//  * try to add an item to a collection that the actor doesn't "own"
	toRun := []tests.RunnableTest{
		tests.TestSuite{
			Name: "control checks",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "target collection is empty",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(target.ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(target.ID),
							tests.IsType(vocab.CollectionType),
							tests.HasTotalItems(0),
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
			Name: "Add side-effects",
			Tests: []tests.RunnableTest{
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

		// NOTE(marius): Undo-ing kept in the same tests
		tests.TestSuite{
			Name:  "Undo Add",
			Tests: []tests.RunnableTest{},
		},
	}

	for _, test := range toRun {
		t.Run(test.Label(), test.Fn(t.Context(), cont))
	}
}
