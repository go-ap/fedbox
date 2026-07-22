//go:build cmds

package integration

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	c "github.com/go-ap/fedbox/integration/internal/containers"
	"github.com/go-ap/fedbox/integration/internal/containers/fedbox"
	"github.com/go-ap/fedbox/integration/internal/tests"
)

func Test_Commands_inSeparateContainers(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	toRun := []tests.CommandTest{
		{
			Name: "--help",
			Host: c2sRootIRI.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"reload"},
				User: c2sRootIRI.String(),
				Key:  privateKey,
			},
			IO: tests.WithTests(tests.EndOK),
		},
		{
			Name: "reload",
			Host: c2sRootIRI.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"reload"},
				User: c2sRootIRI.String(),
				Key:  privateKey,
			},
			IO: tests.WithTests(tests.EndOK),
		},
		{
			Name: "maintenance",
			Host: c2sRootIRI.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"maintenance"},
				User: c2sRootIRI.String(),
				Key:  privateKey,
			},
			IO: tests.WithTests(tests.EndOK),
		},
		{
			Name: "pub actor add",
			Host: c2sRootIRI.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"pub", "actor", "add", "--type", "Person", "--key-type", "RSA", "--tag", "#sysop", "jdoe"},
				User: c2sRootIRI.String(),
				Key:  privateKey,
			},
			IO: tests.WithTests(
				tests.WithInput(tests.PassMatch, "asd"),
				tests.WithInput(tests.ConfirmMatch, "asd"),
				tests.MatchesRegexp(tests.URLRegexp),
				tests.EndOK,
			),
		},
		{
			Name: "oauth client add",
			Host: c2sRootIRI.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"oauth", "client", "add", "--redirect-uri", "http://127.0.0.1"},
				User: c2sRootIRI.String(),
				Key:  privateKey,
			},
			IO: tests.WithTests(
				tests.WithInput(tests.PassMatch, "asd"),
				tests.WithInput(tests.ConfirmMatch, "asd"),
				tests.MatchesRegexp(tests.URLRegexp),
				tests.EndOK),
		},
		{
			Name: "storage bootstrap",
			Host: c2sRootIRI.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"storage", "bootstrap"},
				User: c2sRootIRI.String(),
				Key:  privateKey,
			},
			IO: tests.WithTests(tests.EndOK),
		},
		{
			Name: "password change",
			Host: c2sRootIRI.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"accounts", "pass", c2sRootIRI.String()},
				User: c2sRootIRI.String(),
				Key:  privateKey,
			},
			IO: tests.WithTests(
				tests.WithInput(tests.PassMatch, "asd"),
				tests.WithInput(tests.ConfirmMatch, "asd"),
				tests.EndOK,
			),
		},
		{
			Name: "stop",
			Host: c2sRootIRI.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"stop"},
				User: c2sRootIRI.String(),
				Key:  privateKey,
			},
			IO: tests.WithTests(tests.EndOK),
		},
	}

	for _, test := range toRun {
		t.Run(test.Label(), func(t *testing.T) {
			ctx := t.Context()
			images := c.Suite(fedbox.New(
				fedbox.WithConfig(fedbox.ConfigFromBuildInfo(defaultC2SOptions)),
				fedbox.WithArgs([]string{"--bootstrap"}),
				fedbox.WithImageName(fedBOXImageName),
				fedbox.WithKey(privateKey),
				fedbox.WithRootIRI(c2sRootIRI),
				fedbox.WithPw(rand.Text()[:8]),
				fedbox.WithTestLogger(t, Verbose && test.Label() != "stop"),
			))

			cont, err := c.Init(ctx, t, images...)
			if err != nil {
				t.Fatalf("unable to initialize containers: %s", err)
			}

			var cancelFn func()
			ctx, cancelFn = context.WithTimeout(ctx, 2*time.Second)

			test.Run(ctx, cont, t)
			t.Cleanup(func() {
				if test.Name != "stop" {
					cont.Cleanup(t)
				}
				cancelFn()
			})
		})
	}
}
