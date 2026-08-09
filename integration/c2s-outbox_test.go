//go:build c2s

package integration

import (
	"context"
	"crypto/rand"
	"net/http"
	"testing"
	"time"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/client"
	"github.com/go-ap/client/c2s"
	"github.com/go-ap/client/s2s"
	"github.com/go-ap/errors"
	ap2 "github.com/go-ap/fedbox/activitypub"
	c "github.com/go-ap/fedbox/integration/internal/containers"
	"github.com/go-ap/fedbox/integration/internal/containers/fedbox"
	"github.com/go-ap/fedbox/integration/internal/tests"
	ap "github.com/go-ap/fedbox/integration/internal/vocab"
	"github.com/go-ap/filters"
	"golang.org/x/crypto/ed25519"
)

func Test_C2S_CreateRequests(t *testing.T) {
	_, prvKey, _ := ed25519.GenerateKey(rand.Reader)

	admin := person(
		c2sRootIRI.AddPath("actors/1"),
		ap.HasPreferredUsername("admin"),
		ap.HasAudience(vocab.PublicNS),
	)
	token := new(c2s.BearerSigner)
	rootExec := c.ExecAs(c2sRootIRI, prvKey)
	adminPw := rand.Text()[:8]

	images := c.Suite(fedbox.New(
		fedbox.WithImageName(fedBOXImageName),
		fedbox.WithConfig(fedbox.ConfigFromBuildInfo(defaultC2SOptions)),
		fedbox.WithArgs([]string{"--bootstrap"}),
		fedbox.WithKey(prvKey), fedbox.WithPw(rand.Text()[:8]),
		fedbox.WithTestLogger(t, Verbose),
		fedbox.WithItems(admin),
		fedbox.WithCmd(
			rootExec.SetPassword(admin.ID, adminPw),
			rootExec.ExtractOAuth2Bearer(admin.ID, token),
		),
	))

	ctx := context.Background()
	cont, err := c.Start(ctx, t, images...)
	if err != nil {
		t.Fatalf("Unable to start test containers: %v", err)
	}

	t.Cleanup(func() {
		cont.Cleanup(t)
	})

	person4 := person(
		c2sRootIRI.AddPath("actors/person-4"),
		ap.HasPreferredUsername("example1"),
		ap.HasName("Example McSample"),
		ap.HasReplies,
	)

	article5 := object(
		c2sRootIRI.AddPath("objects/article-5"),
		ap.HasType(vocab.ArticleType),
		ap.HasContent("lorem ipsum dolor sic amet"),
		ap.HasAudience(vocab.PublicNS),
		ap.HasReplies,
	)

	article7 := object(
		c2sRootIRI.AddPath("objects/article-7"),
		ap.HasAudience(vocab.PublicNS),
		ap.HasType(vocab.ArticleType),
		ap.HasContent("this should go to multiple reply collections"),
		ap.HasInReplyTo(article5.ID, person4.ID, c2sRootIRI),
	)

	create3ID := c2sRootIRI.AddPath("activities/create-3")
	create3 := create(ap.HasActor(admin), ap.HasObject(person4))

	create4ID := c2sRootIRI.AddPath("activities/create-4")
	create4 := create(ap.HasActor(admin), ap.HasObject(article5))

	create6ID := c2sRootIRI.AddPath("activities/create-5")
	create6 := create(ap.HasActor(admin), ap.HasObject(article7))

	toRun := []tests.RunnableTest{
		tests.HTTPTest{
			Name: "admin Outbox exists",
			Req: tests.Request().
				Bearer(token.AccessToken).
				Accept(client.ContentTypeJsonActivity).
				IRI(vocab.Outbox.IRI(admin)),
			Res: tests.Response().
				HasCode(http.StatusOK).
				ItemMatch(
					tests.IsType(vocab.OrderedCollectionPageType),
					tests.HasID(filterIRI(vocab.Outbox.IRI(admin), filters.WithMaxCount(filters.MaxItems))),
					tests.HasTotalItems(0),
				),
		},
		tests.TestSuite{
			Name: "Create actor",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Create actor",
					Req: tests.Request().
						Bearer(token.AccessToken).
						IRI(vocab.Outbox.IRI(admin)).
						BodyItem(create3),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						HasLocation(create3ID).
						ItemMatch(
							tests.HasID(person4.ID),
							tests.IsType(person4.Type),
							tests.HasPreferredUsername(person4.PreferredUsername),
							tests.HasName(person4.Name),
							tests.HasInbox(person4.Inbox.GetLink()),
							tests.HasOutbox(person4.Outbox.GetLink()),
							tests.HasReplies(person4.Replies.GetLink()),
						),
				},
				tests.HTTPTest{
					Name: "Create-Actor is in Outbox",
					Req: tests.Request().
						Bearer(token.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(admin)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Outbox.IRI(admin), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(1),
							tests.HasItem(create3ID),
						),
				},
				tests.HTTPTest{
					Name: "Create-Actor is in root Inbox",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Inbox.IRI(c2sRootIRI)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasID(filterIRI(vocab.Inbox.IRI(c2sRootIRI), filters.WithMaxCount(100))),
							tests.HasTotalItems(3),
							tests.HasItem(create3ID),
						),
				},
				tests.HTTPTest{
					Name: "Create is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(create3ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(create3ID),
							tests.IsType(create3.Type),
							tests.HasActor(create3.Actor),
							tests.HasObject(create3.Object),
						),
				},
				tests.HTTPTest{
					Name: "Actor is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(person4.ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(person4.ID),
							tests.IsType(person4.Type),
						),
				},
			},
		},
		tests.TestSuite{
			Name: "Create article",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Create article",
					Req: tests.Request().
						Bearer(token.AccessToken).
						IRI(vocab.Outbox.IRI(admin)).
						BodyItem(create4),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						HasLocation(create4ID).
						ItemMatch(
							tests.HasID(article5.ID),
							tests.IsType(article5.Type),
							tests.HasContent(article5.Content),
							tests.HasReplies(vocab.Replies.IRI(article5)),
						),
				},
				tests.HTTPTest{
					Name: "Create-Article is in Outbox",
					Req: tests.Request().
						Bearer(token.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(admin)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Outbox.IRI(admin), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(2),
							tests.HasItem(create4ID),
						),
				},
				tests.HTTPTest{
					Name: "Create is in root Inbox",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Inbox.IRI(c2sRootIRI)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasID(filterIRI(vocab.Inbox.IRI(c2sRootIRI), filters.WithMaxCount(100))),
							tests.HasTotalItems(4),
							tests.HasItem(create4ID),
						),
				},
				tests.HTTPTest{
					Name: "Create is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(create4ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(create4ID),
							tests.IsType(create4.Type),
							tests.HasActor(create4.Actor),
							tests.HasObject(create4.Object),
						),
				},
				tests.HTTPTest{
					Name: "Article is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(article5.ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(article5.ID),
							tests.IsType(article5.Type),
							tests.HasContent(article5.Content),
						),
				},
			},
		},
		tests.TestSuite{
			Name: "Create Note with multiple InReplyTo",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Create article",
					Req: tests.Request().
						Bearer(token.AccessToken).
						IRI(vocab.Outbox.IRI(admin)).
						BodyItem(create6),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						HasLocation(create6ID).
						ItemMatch(
							tests.HasID(article7.ID),
							tests.IsType(article7.Type),
							tests.HasContent(article7.Content),
						),
				},
				tests.HTTPTest{
					Name: "Create-Article is in Outbox",
					Req: tests.Request().
						Bearer(token.AccessToken).
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Outbox.IRI(admin)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Outbox.IRI(admin), filters.WithMaxCount(filters.MaxItems))),
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasTotalItems(3),
							tests.HasItem(create6ID),
						),
				},
				tests.HTTPTest{
					Name: "Create is in root Inbox",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Inbox.IRI(c2sRootIRI)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.IsType(vocab.OrderedCollectionPageType),
							tests.HasID(filterIRI(vocab.Inbox.IRI(c2sRootIRI), filters.WithMaxCount(100))),
							tests.HasTotalItems(5),
							tests.HasItem(create6ID),
						),
				},
				tests.HTTPTest{
					Name: "Create is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(create6ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(create6ID),
							tests.IsType(create4.Type),
							tests.HasActor(admin.ID),
							tests.HasObject(article7.ID),
						),
				},
				tests.HTTPTest{
					Name: "Article is accessible",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(article7.ID),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(article7.ID),
							tests.IsType(article7.Type),
							tests.HasContent(article7.Content),
							tests.HasInReplyTo(article5.ID, person4.ID, c2sRootIRI),
						),
				},
				tests.HTTPTest{
					Name: "article-5 has article-7 in replies",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Replies.IRI(article5)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Replies.IRI(article5), filters.WithMaxCount(filters.MaxItems))),
							tests.HasTotalItems(1),
							tests.HasItem(article7.ID),
						),
				},
				tests.HTTPTest{
					Name: "person-4 has article-7 in replies",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Replies.IRI(person4)),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(filterIRI(vocab.Replies.IRI(person4), filters.WithMaxCount(filters.MaxItems))),
							tests.HasTotalItems(1),
							tests.HasItem(article7.ID),
						),
				},
				tests.HTTPTest{
					Name: "admin does not have a replies collection",
					Req: tests.Request().
						Accept(client.ContentTypeJsonActivity).
						IRI(vocab.Replies.IRI(admin)),
					Res: tests.Response().
						HasCode(http.StatusNotFound),
				},
			},
		},
	}

	for _, test := range toRun {
		t.Run(test.Label(), test.Fn(ctx, cont))
	}
}

func Test_C2S_Requests(t *testing.T) {
	publicKey, prvKey, _ := ed25519.GenerateKey(rand.Reader)

	service := root(c2sRootIRI, ap.HasPublicKey(publicKey))

	tagAdmin := object(
		c2sRootIRI.AddPath("objects/0"),
		ap.HasName("#sysop"),
		ap.HasTo(vocab.PublicNS),
	)
	admin := person(
		c2sRootIRI.AddPath("actors/1"),
		ap.HasPreferredUsername("admin"),
		ap.HasTag(tagAdmin),
	)

	draftSig := s2s.New(s2s.WithActor(service, prvKey))
	token := new(c2s.BearerSigner)

	toRun := []tests.RunnableTest{
		tests.HTTPTest{
			Name: "tag admin",
			Req:  tests.Request().IRI(tagAdmin.ID),
			Res: tests.Response().
				HasCode(http.StatusOK).
				HasContentType(client.ContentTypeJsonLD).
				ItemMatch(
					tests.HasID(tagAdmin.ID),
					tests.IsType(tagAdmin.Type),
					tests.HasName(tagAdmin.Name),
					tests.HasTo(tagAdmin.To),
				),
		},
		tests.HTTPTest{
			Name: "admin",
			Req:  tests.Request().IRI(admin.ID),
			Res: tests.Response().
				HasCode(http.StatusOK).
				HasContentType(client.ContentTypeJsonLD).
				ItemMatch(
					tests.HasID(admin.ID),
					tests.IsType(admin.Type),
					tests.HasAttributedTo(admin.ID),
					tests.HasTag(admin.Tag),
					tests.HasPreferredUsername(admin.PreferredUsername),
					tests.HasSharedInbox(admin.Endpoints.SharedInbox),
					tests.HasProxyURL(admin.Endpoints.ProxyURL),
					// NOTE(marius): extra added properties
					tests.HasTo(vocab.ItemCollection{vocab.PublicNS}),
				),
		},
		tests.HTTPTest{
			Name: "invalid body",
			Req: tests.Request().IRI(admin.Inbox.GetLink()).
				Post().
				Header("Content-Type", client.ContentTypeJsonLD).
				BodyBytes(nil),
			Res: tests.Response().
				HasCode(http.StatusBadRequest).
				HasContentType(client.ContentTypeJson).
				HasErrors(errors.BadRequestf("unable to unmarshal JSON request")),
		},
		tests.HTTPTest{
			Name: "non authorized",
			Req: tests.Request().IRI(admin.Inbox.GetLink()).
				Post().
				Header("Content-Type", client.ContentTypeJsonLD).
				BodyBytes([]byte(`{"type":"Flag"}`)),
			Res: tests.Response().
				HasCode(http.StatusUnauthorized).
				HasContentType(client.ContentTypeJson).
				HasErrors(errors.Unauthorizedf("unable to read request body")),
		},
		tests.HTTPTest{
			Name: "collection not found",
			Req: tests.Request().IRI(c2sRootIRI.AddPath("test")).
				Post().
				Header("Content-Type", client.ContentTypeJsonLD).
				BodyBytes([]byte(`{"type":"Flag"}`)),
			Res: tests.Response().
				HasCode(http.StatusNotFound).
				HasContentType(client.ContentTypeJson).
				HasErrors(errors.NotFoundf("invalid collection")),
		},
		tests.HTTPTest{
			Name: "to inbox",
			Req: tests.Request().IRI(admin.Inbox.GetLink()).
				Post().
				ContentType(client.ContentTypeJsonLD).
				Header("Date", MockDate.Format(http.TimeFormat)).
				Signer(draftSig.SignDraft).
				BodyBytes([]byte(`{"type":"Flag"}`)),
			Res: tests.Response().
				HasCode(http.StatusBadRequest).
				HasContentType(client.ContentTypeJson).
				HasErrors(
					errors.NewBadRequest(errors.BadRequestf("Activity is not valid: invalid activity id"), "Unable to save activity Flag to http://primary.localdomain/actors/1/inbox"),
					errors.BadRequestf("Activity is not valid: invalid activity id"),
				),
		},
		tests.TestSuite{
			Name: "create to outbox",
			Tests: []tests.RunnableTest{
				tests.CommandTest{
					// NOTE(marius): this command generates an activity and increases object count to 2
					Name: "create OAuth2 token",
					Host: string(c2sRootIRI),
					Cmd: c.SSHCmd{
						Cmd:  []string{"oauth", "token", "add", string(admin.ID)},
						User: c2sRootIRI.String(),
						Key:  prvKey,
					},
					IO: tests.WithTests(tests.GetToken(token), tests.AnyOutput),
				},
				tests.HTTPTest{
					Name: "Create",
					Req: tests.Request().IRI(admin.Outbox.GetLink()).
						Post().
						ContentType(client.ContentTypeJsonLD).
						Signer(token.Sign).
						BodyItem(ap2.WrapObjectInCreate(
							vocab.Object{
								Type:      vocab.NoteType,
								Content:   vocab.DefaultNaturalLanguage("test"),
								Published: MockDate,
							}, admin,
						)),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						HasLocation(c2sRootIRI.AddPath("activities/create-4")).
						ItemMatch(
							tests.HasID(c2sRootIRI.AddPath("objects/note-5")),
							tests.IsType(vocab.NoteType),
							tests.HasContent("test"),
							tests.WasPublished(MockDate),
							tests.HasTo(vocab.ItemCollection{vocab.PublicNS}),
						),
				},
				tests.HTTPTest{
					Name: "check activity",
					Req: tests.Request().
						ContentType(client.ContentTypeJsonLD).
						IRI(c2sRootIRI.AddPath("activities/create-4")),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(c2sRootIRI.AddPath("activities/create-4")),
							tests.IsType(vocab.CreateType),
							tests.HasActor(admin.ID),
							tests.HasObject(c2sRootIRI.AddPath("objects/note-5")),
						),
				},
				tests.HTTPTest{
					Name: "check object",
					Req: tests.Request().
						Signer(token.Sign).
						ContentType(client.ContentTypeJsonLD).
						IRI(c2sRootIRI.AddPath("objects/note-5")),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(c2sRootIRI.AddPath("objects/note-5")),
							tests.IsType(vocab.NoteType),
							tests.HasContent("test"),
							tests.WasPublished(MockDate),
						),
				},
			},
		},
		tests.TestSuite{
			Name: "update to outbox",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					Name: "Update",
					Req: tests.Request().IRI(admin.Outbox.GetLink()).
						Post().
						ContentType(client.ContentTypeJsonLD).
						Signer(token.Sign).
						BodyItem(ap2.WrapObjectInUpdate(
							&vocab.Object{
								ID:      c2sRootIRI.AddPath("objects/note-5"),
								Type:    vocab.NoteType,
								Content: vocab.DefaultNaturalLanguage("updated"),
							}, admin),
						),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						ItemMatch(
							tests.HasID(c2sRootIRI.AddPath("activities/update-6")),
							tests.IsType(vocab.UpdateType),
							tests.WasPublished(time.Now().Round(0)),
							tests.HasTo(vocab.ItemCollection{vocab.PublicNS}),
							tests.HasActor(admin.ID),
							tests.HasObject(c2sRootIRI.AddPath("objects/note-5")),
						),
				},
				tests.HTTPTest{
					Name: "check update activity",
					Req: tests.Request().
						ContentType(client.ContentTypeJsonLD).
						IRI(c2sRootIRI.AddPath("activities/update-6")),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(c2sRootIRI.AddPath("activities/update-6")),
							tests.IsType(vocab.UpdateType),
							tests.WasPublished(time.Now().Round(0)),
							tests.HasTo(vocab.ItemCollection{vocab.PublicNS}),
							tests.HasActor(admin.ID),
							tests.HasObject(c2sRootIRI.AddPath("objects/note-5")),
						),
				},
				tests.HTTPTest{
					Name: "check updated object",
					Req: tests.Request().
						Signer(token.Sign).
						ContentType(client.ContentTypeJsonLD).
						IRI(c2sRootIRI.AddPath("objects/note-5")),
					Res: tests.Response().
						HasCode(http.StatusOK).
						ItemMatch(
							tests.HasID(c2sRootIRI.AddPath("objects/note-5")),
							tests.IsType(vocab.NoteType),
							tests.HasContent("updated"),
							tests.WasPublished(MockDate),
							//tests.WasUpdated(time.Now().Round(0)), // TODO(marius): this doesn't seem to get updated
						),
				},
			},
		},
		tests.TestSuite{
			Name: "Flag activity",
			Tests: []tests.RunnableTest{
				tests.HTTPTest{
					// NOTE(marius): due to previous tests, the cnt is at 5, so our activity is flag-5
					Name: "flag to outbox",
					Req: tests.Request().IRI(admin.Outbox.GetLink()).
						Post().
						ContentType(client.ContentTypeJsonLD).
						Signer(token.Sign).
						BodyItem(&vocab.Activity{
							Type:      vocab.FlagType,
							Actor:     admin.ID,
							Object:    admin.ID,
							Published: MockDate,
						}),
					Res: tests.Response().
						HasCode(http.StatusCreated).
						ItemMatch(
							tests.HasID(c2sRootIRI.AddPath("activities/flag-7")),
							tests.IsType(vocab.FlagType),
							tests.HasActor(admin.ID),
							tests.HasObject(admin.ID),
							tests.WasPublished(MockDate),
						),
				},
				tests.HTTPTest{
					Name: "check flag activity",
					Req: tests.Request().
						Bearer(token.AccessToken).
						ContentType(client.ContentTypeJsonLD).
						IRI(c2sRootIRI.AddPath("activities/flag-7")),
					// TODO(marius): check if this is due to the actor that fetches the flag
					//  is the same actor that has been flagged.
					Res: tests.Response().
						HasCode(http.StatusNotFound).
						HasContentType(client.ContentTypeJson).
						HasErrors(errors.NotFoundf("http://primary.localdomain/activities/flag-7 was not found")),
					//Res: tests.Response().
					//	HasCode(http.StatusOK).
					//	ItemMatch(
					//		tests.HasID(c2sRootIRI.AddPath("activities/update-6")),
					//		tests.IsType(vocab.UpdateType),
					//		tests.WasPublished(time.Now().Round(0)),
					//		tests.HasTo(vocab.ItemCollection{vocab.PublicNS}),
					//		tests.HasActor(admin.ID),
					//		tests.HasObject(admin.ID),
					//	),
				},
			},
		},
	}

	images := c.Suite(fedbox.New(
		fedbox.WithImageName(fedBOXImageName),
		fedbox.WithConfig(fedbox.ConfigFromBuildInfo(defaultC2SOptions)),
		fedbox.WithArgs([]string{"--bootstrap"}),
		fedbox.WithKey(prvKey), fedbox.WithPw(rand.Text()[:8]),
		fedbox.WithItems(tagAdmin, admin),
		fedbox.WithTestLogger(t, Verbose),
	))

	ctx := context.Background()
	cont, err := c.Start(ctx, t, images...)
	if err != nil {
		t.Fatalf("Error: %s", err)
	}

	t.Cleanup(func() {
		cont.Cleanup(t)
	})

	for _, test := range toRun {
		t.Run(test.Label(), test.Fn(ctx, cont))
	}
}
