package integration

import (
	"bytes"
	"context"
	"regexp"
	"testing"
	"time"

	c "github.com/go-ap/fedbox/integration/internal/containers"
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

var urlRegexp = regexp.MustCompile(`(http|https://[a-zA-Z0-9./-]+)`)
var passRegexp = regexp.MustCompile(`.* password:`)
var passAgain = regexp.MustCompile(`password again:`)

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

func Test_Commands_inSeparateContainers(t *testing.T) {
	toRun := []tests.CommandTest{
		{
			Name: "--help",
			Host: service.ID.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"reload"},
				User: service.ID.String(),
				Key:  defaultPrivateKey,
			},
			// NOTE(marius): this is strange. The output should actually be the
			IO: tests.WithTests(endOK),
		},
		{
			Name: "reload",
			Host: service.ID.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"reload"},
				User: service.ID.String(),
				Key:  defaultPrivateKey,
			},
			IO: tests.WithTests(endOK),
		},
		{
			Name: "maintenance",
			Host: service.ID.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"maintenance"},
				User: service.ID.String(),
				Key:  defaultPrivateKey,
			},
			IO: tests.WithTests(endOK),
		},
		{
			Name: "pub actor add",
			Host: service.ID.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"pub", "actor", "add", "--type", "Person", "--key-type", "RSA", "--tag", "#sysop", "jdoe"},
				User: service.ID.String(),
				Key:  defaultPrivateKey,
			},
			IO: tests.WithTests(
				withInput(matchesRegexp(passRegexp), "asd"),
				withInput(matchesRegexp(passAgain), "asd"),
				matchesRegexp(urlRegexp),
				endOK,
			),
		},
		{
			Name: "oauth client add",
			Host: service.ID.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"oauth", "client", "add", "--redirect-uri", "http://127.0.0.1"},
				User: service.ID.String(),
				Key:  defaultPrivateKey,
			},
			IO: tests.WithTests(
				withInput(matchesRegexp(passRegexp), "asd"),
				withInput(matchesRegexp(passAgain), "asd"),
				matchesRegexp(urlRegexp),
				endOK),
		},
		{
			Name: "storage bootstrap",
			Host: service.ID.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"storage", "bootstrap"},
				User: service.ID.String(),
				Key:  defaultPrivateKey,
			},
			IO: tests.WithTests(endOK),
		},
		{
			Name: "password change",
			Host: service.ID.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"accounts", "pass", service.ID.String()},
				User: service.ID.String(),
				Key:  defaultPrivateKey,
			},
			IO: tests.WithTests(
				withInput(matchesRegexp(passRegexp), "asd"),
				withInput(matchesRegexp(passAgain), "asd"),
				endOK,
			),
		},
		{
			Name: "stop",
			Host: service.ID.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"stop"},
				User: service.ID.String(),
				Key:  defaultPrivateKey,
			},
			IO: tests.WithTests(endOK),
		},
	}

	for _, test := range toRun {
		t.Run(test.Label(), func(t *testing.T) {
			envType := c.ExtractEnvTagFromBuild()
			var c2sFedBOX = c.C2SfedBOX(
				c.WithEnv(defaultC2SEnv),
				c.WithArgs([]string{"--env", envType, "--bootstrap"}),
				c.WithImageName(fedBOXImageName),
				c.WithKey(defaultPrivateKey),
				c.WithUser(service.ID),
				c.WithPw(defaultPassword),
				c.WithTestLogger(Verbose, t),
			)
			ctx := context.Background()

			images := c.Suite{c2sFedBOX}
			cont, err := c.Init(ctx, t, images...)
			if err != nil {
				t.Fatalf("unable to initialize containers: %s", err)
			}

			var cancelFn func()
			ctx, cancelFn = context.WithTimeout(ctx, 2*time.Second)

			test.Run(ctx, cont, t)
			t.Cleanup(func() {
				cancelFn()
				cont.Cleanup(t)
			})
		})
	}
}
