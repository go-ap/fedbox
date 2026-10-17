//go:build c2s || Question

package integration

import (
	"net/http"
	"testing"
	"time"

	vocab "github.com/go-ap/activitypub"
	ap "github.com/go-ap/activitypub/build"
	"github.com/go-ap/client"
	"github.com/go-ap/client/c2s"
	c "github.com/go-ap/fedbox/integration/internal/containers"
	"github.com/go-ap/fedbox/integration/internal/containers/fedbox"
	"github.com/go-ap/fedbox/integration/internal/tests"
	"github.com/go-ap/filters"
)

func Test_C2S_QuestionRequests(t *testing.T) {
	tokenP1 := new(c2s.BearerSigner)
	rootExec := c.ExecAs(c2sRootIRI, ed2559Key)

	conf := fedbox.C2SConfig(
		fedbox.WithImageName(imageName),
		fedbox.WithItems(person1),
		fedbox.WithPrivateKey(ed2559Key),
		fedbox.Verbose(verbose),
		fedbox.WithCommands(rootExec.ExtractOAuth2Bearer(person1.ID, tokenP1)),
		fedbox.Verbose(verbose), fedbox.WithCodeCoverage(coverage),
	)

	cont, err := fedbox.StartContainers(t.Context(), t, conf)
	if err != nil {
		t.Fatalf("Unable to start test containers: %+v", err)
	}

	question3ID := c2sRootIRI.AddPath("/activities/question-3")
	question3 := ap.Question(
		ap.WithName("Some question"),
		ap.WithAudience(vocab.PublicNS),
		ap.WithActor(person1.ID),
	)

	option5ID := c2sRootIRI.AddPath("/activities/note-5")
	opt5 := ap.Note(ap.WithID(option5ID), ap.WithName("Option 1"), ap.WithPublished(MockDate))
	option6ID := c2sRootIRI.AddPath("/activities/note-6")
	opt6 := ap.Note(ap.WithID(option6ID), ap.WithName("Option 2"), ap.WithPublished(MockDate))

	question4ID := c2sRootIRI.AddPath("/activities/question-4")
	question4 := ap.Question(
		ap.WithName("A second question, any answer"),
		ap.WithActor(person1.ID),
		ap.WithTo(vocab.PublicNS),
		ap.WithAnyOf(opt5, opt6),
	)

	option7ID := c2sRootIRI.AddPath("/activities/note-7")
	opt7 := ap.Note(ap.WithID(option7ID), ap.WithName("Option 1"), ap.WithPublished(MockDate))
	option8ID := c2sRootIRI.AddPath("/activities/note-8")
	opt8 := ap.Note(ap.WithID(option8ID), ap.WithName("Option 2"), ap.WithPublished(MockDate))
	option9ID := c2sRootIRI.AddPath("/activities/note-9")
	opt9 := ap.Note(ap.WithID(option9ID), ap.WithName("Option 3"), ap.WithPublished(MockDate))
	question5ID := c2sRootIRI.AddPath("/activities/question-5")
	question5 := ap.Question(
		ap.WithName("A third question, only one answer"),
		ap.WithActor(person1.ID),
		ap.WithAudience(vocab.PublicNS),
		ap.WithOneOf(opt7, opt8, opt9),
	)

	toRun := []tests.RunnableTest{
		tests.TestSuite{
			Name: "Question w/o answers",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Question 3",
					Req: tests.Request().
						Bearer(tokenP1.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1)).
						BodyItem(question3),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						ItemMatch(
							tests.HasID(question3ID),
							tests.IsType(question3.Type),
							tests.HasContent(question3.Content),
							tests.HasActor(question3.Actor),
							tests.HasAnyOf(question3.AnyOf),
							tests.HasOneOf(question3.OneOf),
							tests.WasPublished(time.Now()),
						),
				},
				tests.HTTPTest{
					Name: "Question3 is in person1's Outbox",
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
							tests.HasItem(question3ID),
						),
				},
				tests.HTTPTest{
					Name: "Question3 is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(question3ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(question3ID),
							tests.IsType(question3.Type),
							tests.HasActor(question3.Actor),
							tests.HasContent(question3.Content),
							tests.WasPublished(time.Now().Round(0)),
							tests.HasAnyOf(question3.AnyOf),
							tests.HasOneOf(question3.OneOf),
						),
				},
			},
		},

		tests.TestSuite{
			Name: "Question with anyOf answers",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Question 4",
					Req: tests.Request().
						Bearer(tokenP1.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1)).
						BodyItem(question4),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						ItemMatch(
							tests.HasID(question4ID),
							tests.IsType(question4.Type),
							tests.HasContent(question4.Content),
							tests.HasActor(question4.Actor),
							tests.HasAnyOf(
								ap.Note(ap.WithID(option5ID), ap.WithName("Option 1"), ap.WithTo(vocab.PublicNS), ap.WithAttributedTo(person1.ID), ap.WithPublished(MockDate)),
								ap.Note(ap.WithID(option6ID), ap.WithName("Option 2"), ap.WithTo(vocab.PublicNS), ap.WithAttributedTo(person1.ID), ap.WithPublished(MockDate)),
							),
							tests.WasPublished(time.Now()),
						),
				},
				tests.HTTPTest{
					Name: "Question4 is in person1's Outbox",
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
							tests.HasItem(question3ID),
							tests.HasItem(question4ID),
						),
				},
				tests.HTTPTest{
					Name: "Question4 is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(question4ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(question4ID),
							tests.IsType(question4.Type),
							tests.HasActor(question4.Actor),
							tests.HasContent(question4.Content),
							tests.WasPublished(time.Now().Round(0)),
							tests.HasAnyOf(
								ap.Note(ap.WithID(option5ID), ap.WithName("Option 1"), ap.WithTo(vocab.PublicNS), ap.WithAttributedTo(person1.ID), ap.WithPublished(MockDate)),
								ap.Note(ap.WithID(option6ID), ap.WithName("Option 2"), ap.WithTo(vocab.PublicNS), ap.WithAttributedTo(person1.ID), ap.WithPublished(MockDate)),
							),
						),
				},
			},
		},

		tests.TestSuite{
			Name: "Question with oneOf answers",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Question 5",
					Req: tests.Request().
						Bearer(tokenP1.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1)).
						BodyItem(question5),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						ItemMatch(
							tests.HasID(question5ID),
							tests.IsType(question5.Type),
							tests.HasContent(question5.Content),
							tests.HasActor(question5.Actor),
							tests.HasOneOf(
								ap.Note(ap.WithID(option7ID), ap.WithName("Option 1"), ap.WithAudience(vocab.PublicNS), ap.WithAttributedTo(person1.ID), ap.WithPublished(MockDate)),
								ap.Note(ap.WithID(option8ID), ap.WithName("Option 2"), ap.WithAudience(vocab.PublicNS), ap.WithAttributedTo(person1.ID), ap.WithPublished(MockDate)),
								ap.Note(ap.WithID(option9ID), ap.WithName("Option 3"), ap.WithAudience(vocab.PublicNS), ap.WithAttributedTo(person1.ID), ap.WithPublished(MockDate)),
							),
							tests.WasPublished(time.Now()),
						),
				},
				tests.HTTPTest{
					Name: "Question5 is in person1's Outbox",
					Req: tests.Request().
						Bearer(tokenP1.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(person1.ID)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Outbox.IRI(person1.ID), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(3),
							tests.HasItem(question3ID),
							tests.HasItem(question4ID),
							tests.HasItem(question5ID),
						),
				},
				tests.HTTPTest{
					Name: "Question5 is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(question5ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(question5ID),
							tests.IsType(question5.Type),
							tests.HasActor(question5.Actor),
							tests.HasContent(question5.Content),
							tests.WasPublished(time.Now().Round(0)),
							tests.HasOneOf(
								ap.Note(ap.WithID(option7ID), ap.WithName("Option 1"), ap.WithAudience(vocab.PublicNS), ap.WithAttributedTo(person1.ID), ap.WithPublished(MockDate)),
								ap.Note(ap.WithID(option8ID), ap.WithName("Option 2"), ap.WithAudience(vocab.PublicNS), ap.WithAttributedTo(person1.ID), ap.WithPublished(MockDate)),
								ap.Note(ap.WithID(option9ID), ap.WithName("Option 3"), ap.WithAudience(vocab.PublicNS), ap.WithAttributedTo(person1.ID), ap.WithPublished(MockDate)),
							),
						),
				},
			},
		},
	}

	for _, test := range toRun {
		t.Run(test.Label(), test.Fn(t.Context(), cont))
	}
}
