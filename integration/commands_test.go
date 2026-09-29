//go:build ssh

package integration

import (
	"bytes"
	"net/http"
	"strconv"
	"testing"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/client"
	c "github.com/go-ap/fedbox/integration/internal/containers"
	"github.com/go-ap/fedbox/integration/internal/containers/fedbox"
	"github.com/go-ap/fedbox/integration/internal/tests"
)

func Test_Commands(t *testing.T) {
	conf := fedbox.C2SConfig(
		fedbox.WithImageName(imageName),
		fedbox.WithPrivateKey(ed2559Key),
		fedbox.Verbose(verbose), fedbox.WithCodeCoverage(coverage),
	)
	cont, err := fedbox.StartContainers(t.Context(), t, conf)
	if err != nil {
		t.Fatalf("Error: %s", err)
	}

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
				Key:  ed2559Key,
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
						Key:  ed2559Key,
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
						Key:  ed2559Key,
					},
					IO: tests.WithTests(
						tests.WithInput(tests.PassMatch, "asd"),
						tests.WithInput(tests.ConfirmMatch, "asd"),
						tests.ExtractActorIRI(clientIRI),
						tests.EndOK),
				},
				tests.HTTPTest{
					Name: "check actor iri",
					Req:  tests.Request().IRIFunc(func() vocab.IRI { return *clientIRI }),
					Res: tests.Response().
						HasCode(http.StatusOK).
						HasContentType(client.ContentTypeJsonLD).
						ItemMatch(
							tests.IsType(vocab.ApplicationType),
							tests.HasURL(vocab.IRI("http://127.0.0.1")),
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
				Key:  ed2559Key,
			},
			IO: tests.WithTests(tests.MatchToken, tests.EndOK),
		},
		tests.CommandTest{
			Name: "password change",
			Host: string(c2sRootIRI),
			Cmd: c.SSHCmd{
				Cmd:  []string{"accounts", "pass", string(c2sRootIRI)},
				User: string(c2sRootIRI),
				Key:  ed2559Key,
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
				Key:  ed2559Key,
			},
			IO: tests.WithTests(tests.EndOK),
		},
		tests.CommandTest{
			Name: "gen-keys root actor",
			Host: string(c2sRootIRI),
			Cmd: c.SSHCmd{
				Cmd:  []string{"accounts", "gen-keys", string(c2sRootIRI)},
				User: string(c2sRootIRI),
				Key:  ed2559Key,
			},
			IO: tests.WithTests(tests.EndOK),
		},
	}

	for _, test := range toRun {
		t.Run(test.Label(), test.Fn(t.Context(), cont))
	}
}

func Test_Commands_Import(t *testing.T) {
	conf := fedbox.C2SConfig(
		fedbox.WithImageName(imageName),
		fedbox.WithPrivateKey(ed2559Key),
		fedbox.Verbose(verbose), fedbox.WithCodeCoverage(coverage),
	)

	items := plausibleRandomObjects(ed2559Key.Public(), 60)
	conf.InitFns = append(conf.InitFns, c.WithMocks(items...))
	cont, err := fedbox.StartContainers(t.Context(), t, conf)
	if err != nil {
		t.Fatalf("Error: %s", err)
	}

	outputCheckFns := make([]tests.LineOutputTest, 0, len(items)+2)
	for range items {
		outputCheckFns = append(outputCheckFns, tests.AnyOutput)
	}
	outputCheckFns = append(outputCheckFns, func(tb testing.TB, raw []byte) []byte {
		lines := bytes.Split(raw, []byte{'\n'})
		{
			line := lines[0]
			ok := []byte("Activities count:                     " + strconv.Itoa(len(items)))
			if !bytes.Equal(line, ok) {
				t.Errorf("Output line %q, expected: %q", line, ok)
			}
		}
		{
			line := lines[1]
			ok := []byte("Activities processing time:")
			if !bytes.HasPrefix(lines[1], ok) {
				t.Errorf("Output line %q, expected: %q", line, ok)
			}
		}
		{
			line := lines[2]
			ok := []byte("Elapsed time per activity:")
			if !bytes.HasPrefix(line, ok) {
				t.Errorf("Output line %q, expected: %q", line, ok)
			}
		}
		{
			line := lines[3]
			ok := []byte("Import done!")
			if !bytes.Equal(line, ok) {
				t.Errorf("Output line %q, expected: %q", line, ok)
			}
		}
		return nil
	})

	tests := []tests.RunnableTest{
		tests.CommandTest{
			Name: "import(ssh)",
			Host: string(c2sRootIRI),
			Cmd: c.SSHCmd{
				Cmd:  []string{"pub", "import", "--skip-remotes", "/storage/import.json"},
				User: string(c2sRootIRI),
				Key:  ed2559Key,
			},
			IO: tests.WithTests(append(outputCheckFns, tests.EndOK)...),
		},
		//tests.CommandTest{
		//	Name: "import(cmd)",
		//	Host: string(c2sRootIRI),
		//	Cmd:  tc.NewRawCommand([]string{"fedbox", "pub", "import", "--skip-remotes", "/storage/import.json"}),
		//	IO:   tests.WithTests(outputCheckFns...),
		//},
	}
	for _, tt := range tests {
		tt.Run(t.Context(), cont, t)
	}
}

func Test_Commands_Maintenance(t *testing.T) {
	conf := fedbox.C2SConfig(
		fedbox.WithImageName(imageName),
		fedbox.WithPrivateKey(ed2559Key),
		fedbox.Verbose(verbose), fedbox.WithCodeCoverage(coverage),
	)
	cont, err := fedbox.StartContainers(t.Context(), t, conf)
	if err != nil {
		t.Fatalf("Error: %s", err)
	}

	tests.CommandTest{
		Name: "maintenance",
		Host: string(c2sRootIRI),
		Cmd: c.SSHCmd{
			Cmd:  []string{"maintenance"},
			User: string(c2sRootIRI),
			Key:  ed2559Key,
		},
		IO: tests.WithTests(tests.EndOK),
	}.Run(t.Context(), cont, t)
}

func Test_Commands_Stop(t *testing.T) {
	conf := fedbox.C2SConfig(
		fedbox.WithImageName(imageName),
		fedbox.WithPrivateKey(ed2559Key),
		fedbox.Verbose(verbose), fedbox.WithCodeCoverage(coverage),
	)
	cont, err := fedbox.StartContainers(t.Context(), t, conf)
	if err != nil {
		t.Fatalf("Error: %s", err)
	}

	tests.CommandTest{
		Name: "stop",
		Host: string(c2sRootIRI),
		Cmd: c.SSHCmd{
			Cmd:  []string{"stop"},
			User: string(c2sRootIRI),
			Key:  ed2559Key,
		},
		IO: tests.WithTests(tests.EndOK),
	}.Run(t.Context(), cont, t)
}
