//go:build ssh

package integration

import (
	"net/http"
	"testing"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/client"
	c "github.com/go-ap/fedbox/integration/internal/containers"
	"github.com/go-ap/fedbox/integration/internal/tests"
)

func Test_Commands(t *testing.T) {
	ctx := t.Context()
	_, privateKey, cont, err := initC2SContainers(ctx, t)
	if err != nil {
		t.Fatalf("Error: %s", err)
	}

	t.Cleanup(func() {
		cont.Cleanup(t)
	})

	clientIRI := new(vocab.IRI)
	var actorIRI vocab.IRI
	toRun := []tests.RunnableTest{
		//tests.CommandTest{
		//	Name: "--help",
		//	Host: string(c2sRootIRI),
		//	Cmd: c.SSHCmd{
		//		Cmd:  []string{"--help"},
		//		User: string(c2sRootIRI),
		//		Key:  privateKey,
		//	},
		//	// NOTE(marius): this is strange, the help should be a single buffer output, not 4
		//	// So we disabled this for the moment
		//	IO: tests.WithTests(tests.AnyOutput, tests.AnyOutput, tests.AnyOutput, tests.AnyOutput),
		//},
		tests.CommandTest{
			Name: "reload",
			Host: string(c2sRootIRI),
			Cmd: c.SSHCmd{
				Cmd:  []string{"reload"},
				User: string(c2sRootIRI),
				Key:  privateKey,
			},
			IO: tests.WithTests(tests.EndOK),
		},
		tests.TestSuite{
			Name: "actor add",
			Tests: []tests.RunnableTest{
				tests.CommandTest{
					Name: "pub actor add",
					Host: string(c2sRootIRI),
					Cmd: c.SSHCmd{
						Cmd:  []string{"pub", "actor", "add", "--type", "Person", "--key-type", "RSA", "--tag", "#sysop", "jdoe"},
						User: string(c2sRootIRI),
						Key:  privateKey,
					},
					IO: tests.WithTests(
						tests.WithInput(tests.PassMatch, "asd"),
						tests.WithInput(tests.ConfirmMatch, "asd"),
						tests.ExtractActorIRI(&actorIRI),
						tests.EndOK,
					),
				},
				tests.HTTPTest{
					Name: "check actor iri",
					Req: tests.Request().IRIFunc(func() vocab.IRI {
						return actorIRI
					}),
					Res: tests.Response().
						HasCode(http.StatusOK).
						HasContentType(client.ContentTypeJsonLD).
						ItemMatch(
							tests.IsType(vocab.PersonType),
							tests.HasPreferredUsername("jdoe"),
						),
				},
			},
		},
		tests.TestSuite{
			Name: "client add",
			Tests: []tests.RunnableTest{
				tests.CommandTest{
					Name: "oauth client add",
					Host: string(c2sRootIRI),
					Cmd: c.SSHCmd{
						Cmd:  []string{"oauth", "client", "add", "--redirect-uri", "http://127.0.0.1"},
						User: string(c2sRootIRI),
						Key:  privateKey,
					},
					IO: tests.WithTests(
						tests.WithInput(tests.PassMatch, "asd"),
						tests.WithInput(tests.ConfirmMatch, "asd"),
						tests.ExtractActorIRI(clientIRI),
						tests.EndOK),
				},
				tests.HTTPTest{
					Name: "check actor iri",
					Req: tests.Request().IRIFunc(func() vocab.IRI {
						return *clientIRI
					}),
					Res: tests.Response().
						HasCode(http.StatusOK).
						HasContentType(client.ContentTypeJsonLD).
						ItemMatch(
							tests.IsType(vocab.ApplicationType),
							tests.HasURL("http://127.0.0.1"),
						),
				},
			},
		},
		tests.CommandTest{
			Name: "oauth token generate",
			Host: string(c2sRootIRI),
			Cmd: c.SSHCmd{
				Cmd:  []string{"oauth", "token", "add", string(c2sRootIRI)},
				User: string(c2sRootIRI),
				Key:  privateKey,
			},
			IO: tests.WithTests(tests.MatchToken, tests.EndOK),
		},
		tests.CommandTest{
			Name: "password change",
			Host: string(c2sRootIRI),
			Cmd: c.SSHCmd{
				Cmd:  []string{"accounts", "pass", string(c2sRootIRI)},
				User: string(c2sRootIRI),
				Key:  privateKey,
			},
			IO: tests.WithTests(
				tests.WithInput(tests.PassMatch, "asd"),
				tests.WithInput(tests.ConfirmMatch, "asd"),
				tests.EndOK,
			),
		},
		tests.CommandTest{
			Name: "gen-keys all",
			Host: string(c2sRootIRI),
			Cmd: c.SSHCmd{
				Cmd:  []string{"accounts", "gen-keys"},
				User: string(c2sRootIRI),
				Key:  privateKey,
			},
			IO: tests.WithTests(tests.EndOK),
		},
		tests.CommandTest{
			Name: "gen-keys root actor",
			Host: string(c2sRootIRI),
			Cmd: c.SSHCmd{
				Cmd:  []string{"accounts", "gen-keys", string(c2sRootIRI)},
				User: string(c2sRootIRI),
				Key:  privateKey,
			},
			IO: tests.WithTests(tests.EndOK),
		},
	}

	for _, test := range toRun {
		t.Run(test.Label(), test.Fn(ctx, cont))
	}
}

func Test_Commands_StorageBootstrap(t *testing.T) {
	ctx := t.Context()

	_, privateKey, cont, err := initC2SContainers(ctx, t)
	if err != nil {
		t.Fatalf("Error: %s", err)
	}
	t.Cleanup(func() {
		cont.Cleanup(t)
	})

	tests.CommandTest{
		Name: "storage bootstrap",
		Host: string(c2sRootIRI),
		Cmd: c.SSHCmd{
			Cmd:  []string{"storage", "bootstrap"},
			User: string(c2sRootIRI),
			Key:  privateKey,
		},
		IO: tests.WithTests(tests.EndOK),
	}.Run(ctx, cont, t)
}

func Test_Commands_Maintenance(t *testing.T) {
	ctx := t.Context()

	_, privateKey, cont, err := initC2SContainers(ctx, t)
	if err != nil {
		t.Fatalf("Error: %s", err)
	}
	t.Cleanup(func() {
		cont.Cleanup(t)
	})

	tests.CommandTest{
		Name: "maintenance",
		Host: string(c2sRootIRI),
		Cmd: c.SSHCmd{
			Cmd:  []string{"maintenance"},
			User: string(c2sRootIRI),
			Key:  privateKey,
		},
		IO: tests.WithTests(tests.EndOK),
	}.Run(ctx, cont, t)
}

func Test_Commands_Stop(t *testing.T) {
	ctx := t.Context()

	_, privateKey, cont, err := initC2SContainers(ctx, t)
	if err != nil {
		t.Fatalf("Error: %s", err)
	}
	t.Cleanup(func() {
		cont.Cleanup(t)
	})

	tests.CommandTest{
		Name: "stop",
		Host: string(c2sRootIRI),
		Cmd: c.SSHCmd{
			Cmd:  []string{"stop"},
			User: string(c2sRootIRI),
			Key:  privateKey,
		},
		IO: tests.WithTests(tests.EndOK),
	}.Run(ctx, cont, t)
}
