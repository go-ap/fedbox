//go:build cmds

package integration

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/go-ap/client/c2s"
	c "github.com/go-ap/fedbox/integration/internal/containers"
	"github.com/go-ap/fedbox/integration/internal/containers/fedbox"
	"github.com/go-ap/fedbox/integration/internal/tests"
)

func anyOutput(t *testing.T, line []byte) []byte {
	t.Logf("read %q", bytes.Trim(line, string(tests.CRLF)))
	return nil
}

func endOK(t *testing.T, line []byte) []byte {
	ok := []byte("FedBOX SSH: OK")
	if !bytes.Equal(line, ok) {
		t.Errorf("Output line %q, expected: %q", line, ok)
	}
	return nil
}

func extractToken(token *c2s.BearerSigner) func(t *testing.T, i []byte) []byte {
	return func(t *testing.T, i []byte) []byte {
		i = bytes.TrimSpace(i)
		auth, found := bytes.CutPrefix(i, []byte("Authorization: "))
		if !found {
			t.Fatalf("Unable to get Authorization value from CLI output: %s", i)
		}
		if hasEoL := bytes.IndexByte(auth, '\n'); hasEoL > 0 {
			auth = auth[:hasEoL]
		}
		authPieces := strings.Split(string(auth), " ")
		if len(authPieces) < 2 {
			t.Fatalf("Authorization value is not recognized: %+v", authPieces)
		}
		token.TokenType = strings.TrimSpace(authPieces[0])
		token.AccessToken = strings.TrimSpace(authPieces[1])
		if token.AccessToken == "" || token.TokenType == "" {
			t.Fatalf("Unable to build Authorization token")
		}
		return nil
	}
}

var urlRegexp = regexp.MustCompile(`(http|https://[a-zA-Z0-9./-]+)`)
var passMatch = matchesString(`Password: `)
var confirmMatch = matchesString(` Confirm: `)

func withInput(r tests.LineOutputTest, input string) tests.LineOutputTest {
	return func(t *testing.T, line []byte) []byte {
		// NOTE(marius): ignore any input returned by previous test
		_ = r(t, line)
		return []byte(input)
	}
}

func matchesRegexp(r *regexp.Regexp) tests.LineOutputTest {
	return func(t *testing.T, line []byte) []byte {
		if matches := r.FindSubmatch(line); len(matches) == 0 {
			t.Errorf("The line %q did not contain the regex, %q", line, r)
		}
		return nil
	}
}

func matchesString(s string) tests.LineOutputTest {
	return func(t *testing.T, line []byte) []byte {
		if !bytes.Equal(line, []byte(s)) {
			t.Errorf("The line %q did not match expected, %q", line, s)
		}
		return nil
	}
}

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
			IO: tests.WithTests(endOK),
		},
		{
			Name: "reload",
			Host: c2sRootIRI.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"reload"},
				User: c2sRootIRI.String(),
				Key:  privateKey,
			},
			IO: tests.WithTests(endOK),
		},
		{
			Name: "maintenance",
			Host: c2sRootIRI.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"maintenance"},
				User: c2sRootIRI.String(),
				Key:  privateKey,
			},
			IO: tests.WithTests(endOK),
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
				withInput(passMatch, "asd"),
				withInput(confirmMatch, "asd"),
				matchesRegexp(urlRegexp),
				endOK,
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
				withInput(passMatch, "asd"),
				withInput(confirmMatch, "asd"),
				matchesRegexp(urlRegexp),
				endOK),
		},
		{
			Name: "storage bootstrap",
			Host: c2sRootIRI.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"storage", "bootstrap"},
				User: c2sRootIRI.String(),
				Key:  privateKey,
			},
			IO: tests.WithTests(endOK),
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
				withInput(passMatch, "asd"),
				withInput(confirmMatch, "asd"),
				endOK,
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
			IO: tests.WithTests(endOK),
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
