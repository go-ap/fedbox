package integration

import (
	"bytes"
	"context"
	"io"
	"regexp"
	"testing"

	"github.com/go-ap/errors"
	c "github.com/go-ap/fedbox/integration/internal/containers"
	"github.com/google/go-cmp/cmp"
	tc "github.com/testcontainers/testcontainers-go"
)

var equateWeakErrors = cmp.FilterValues(areErrors, cmp.Comparer(compareErrors))

func areErrors(x, y interface{}) bool {
	_, ok1 := x.(error)
	_, ok2 := y.(error)
	return ok1 && ok2
}

func compareErrors(x, y interface{}) bool {
	xe := x.(error)
	ye := y.(error)
	return errors.Is(xe, ye) || errors.Is(ye, xe) || xe.Error() == ye.Error()
}

func eqErrs(want, got error) bool {
	return cmp.Equal(want, got, equateWeakErrors)
}

func diffErrs(want, got error) string {
	return cmp.Diff(want, got, equateWeakErrors)
}

type testReadWriter struct {
	t           *testing.T
	inLines     [][]byte
	checkOutput []testLineOutput
}

func (t testReadWriter) Read(p []byte) (n int, err error) {
	if len(t.inLines) == 0 {
		t.t.Fatalf("Input was asked for, but no output was provided in the test setup")
		return 0, io.ErrUnexpectedEOF
	}

	line := t.inLines[0]
	t.inLines = t.inLines[1:]
	return copy(p, line), nil
}

func (t testReadWriter) Write(p []byte) (n int, err error) {
	if len(t.checkOutput) == 0 {
		t.t.Fatalf("Output was provided, but no handler was provided in the test setup")
	}
	checker := t.checkOutput[0]
	checker(t.t, p)
	t.checkOutput = t.checkOutput[1:]
	return len(p), nil
}

var _ io.ReadWriter = testReadWriter{}

type testLineOutput func(*testing.T, []byte)

func anyOutput(t *testing.T, line []byte) {
	t.Logf("read %q", line)
}

func endOK(t *testing.T, line []byte) {
	ok := []byte("FedBOX SSH: OK")
	if !bytes.Equal(line, ok) {
		t.Errorf("Output line %q, expected: %q", line, ok)
	}
}

var urlRegexp = regexp.MustCompile(`(http|https://[a-zA-Z0-9./-]+)`)

func matchesRegexp(r *regexp.Regexp) testLineOutput {
	return func(t *testing.T, line []byte) {
		if matches := r.FindSubmatch(line); len(matches) > 1 {
			return
		}
		t.Errorf("The line %q did not contain the regex, %q", line, r)
	}
}

type testerInitFn func(writer *testReadWriter)

func withInput(lines ...[]byte) testerInitFn {
	return func(w *testReadWriter) {
		w.inLines = lines
	}
}
func withTests(testFns ...testLineOutput) testerInitFn {
	return func(w *testReadWriter) {
		w.checkOutput = testFns
	}
}
func tester(t *testing.T, fns ...testerInitFn) *testReadWriter {
	tt := testReadWriter{t: t}
	for _, fn := range fns {
		fn(&tt)
	}
	return &tt
}

func Test_Commands_inSeparateContainers(t *testing.T) {
	toRun := []struct {
		Name    string
		Host    string
		Cmd     tc.Executable
		IO      io.ReadWriter
		WantErr error
	}{
		{
			Name: "--help",
			Host: service.ID.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"reload"},
				User: service.ID.String(),
				Key:  defaultPrivateKey,
			},
			// NOTE(marius): this is strange. The output should actually be the
			IO: tester(t, withTests(endOK)),
		},
		{
			Name: "reload",
			Host: service.ID.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"reload"},
				User: service.ID.String(),
				Key:  defaultPrivateKey,
			},
			IO: tester(t, withTests(endOK)),
		},
		{
			Name: "maintenance",
			Host: service.ID.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"maintenance"},
				User: service.ID.String(),
				Key:  defaultPrivateKey,
			},
			IO: tester(t, withTests(endOK)),
		},
		{
			Name: "pub actor add",
			Host: service.ID.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"pub", "actor", "add", "--type", "Person", "--key-type", "RSA", "--tag", "#sysop", "jdoe"},
				User: service.ID.String(),
				Key:  defaultPrivateKey,
			},
			IO: tester(
				t,
				withInput([]byte("asd\n"), []byte("asd\n")),
				withTests(
					anyOutput,
					anyOutput,
					matchesRegexp(urlRegexp),
					endOK,
				),
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
			IO: tester(t, withInput([]byte("asd\n"), []byte("asd\n")), withTests(
				anyOutput,
				anyOutput,
				matchesRegexp(urlRegexp),
				endOK),
			),
		},
		{
			Name: "storage bootstrap",
			Host: service.ID.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"storage", "bootstrap"},
				User: service.ID.String(),
				Key:  defaultPrivateKey,
			},
			IO: tester(t, withTests(endOK)),
		},
		{
			Name: "password change",
			Host: service.ID.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"accounts", "pass", service.ID.String()},
				User: service.ID.String(),
				Key:  defaultPrivateKey,
			},
			IO: tester(t,
				withInput([]byte("test\n"), []byte("test\n")),
				withTests(
					anyOutput,
					anyOutput,
					endOK,
				),
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
			IO: tester(t, withTests(endOK)),
		},
	}

	for _, test := range toRun {
		t.Run(test.Name, func(t *testing.T) {
			//envType := c.ExtractEnvTagFromBuild()
			var c2sFedBOX = c.C2SfedBOX(
				c.WithEnv(defaultC2SEnv),
				//c.WithArgs([]string{"--env", envType, "--bootstrap"}),
				c.WithImageName(fedBOXImageName),
				c.WithKey(defaultPrivateKey),
				c.WithUser(service.ID),
				c.WithPw(defaultPassword),
			)

			images := c.Suite{c2sFedBOX}

			ctx := context.Background()
			cont, err := c.Init(ctx, t, images...)
			if err != nil {
				t.Fatalf("unable to initialize containers: %s", err)
			}

			t.Cleanup(func() {
				cont.Cleanup(t)
			})
			_, err = cont.RunCommand(ctx, test.Host, test.Cmd, test.IO)
			if !eqErrs(test.WantErr, err) {
				if test.Cmd == nil {
					t.Fatalf("Err received executing nil command %s: %+v", test.Host, diffErrs(test.WantErr, err))
				}
				t.Fatalf("Err received executing command %s->%v: %+v", test.Host, test.Cmd.AsCommand(), diffErrs(test.WantErr, err))
			}
		})
	}
}
