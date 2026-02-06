package integration

import (
	"bytes"
	"context"
	"io"
	"regexp"
	"testing"
	"time"

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
	input       []byte
	checkOutput []testLineOutput
}

func (t *testReadWriter) Read(p []byte) (n int, err error) {
	if len(t.input) == 0 {
		return 0, io.EOF
	}

	if !(bytes.HasSuffix(t.input, []byte{'\n'}) || bytes.HasSuffix(t.input, []byte{'\n', '\r'})) {
		t.input = append(t.input, '\n')
	}
	if len(p) < len(t.input) {
		t.t.Errorf("Input buffer smaller than provided value")
		return 0, io.ErrShortBuffer
	}
	cnt, err := io.Copy(bytes.NewBuffer(p), bytes.NewReader(t.input))
	return int(cnt), err
}

func (t *testReadWriter) Write(p []byte) (n int, err error) {
	t.t.Logf("OUT: %s", p)
	if len(t.checkOutput) == 0 {
		t.t.Errorf("output was provided, but no handler was provided in the test setup")
		return len(p), nil
	}
	lines := bytes.Split(bytes.TrimSpace(p), []byte("\n"))
	for i, line := range lines {
		line = bytes.TrimSpace(p)
		checker := t.checkOutput[i]
		t.input = append(t.input, checker(t.t, line)...)
	}
	if len(t.checkOutput) >= len(lines) {
		t.checkOutput = t.checkOutput[len(lines):]
	}
	return len(p), nil
}

var _ io.ReadWriter = new(testReadWriter)

type testLineOutput func(*testing.T, []byte) []byte

func anyOutput(t *testing.T, line []byte) []byte {
	t.Logf("read %q", line)
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

func withInput(r testLineOutput, input string) testLineOutput {
	return func(t *testing.T, line []byte) []byte {
		// NOTE(marius): ignore any input returned by previous test
		_ = r(t, line)
		return append([]byte(input), []byte("\r\n")...)
	}
}

func matchesRegexp(r *regexp.Regexp) testLineOutput {
	return func(t *testing.T, line []byte) []byte {
		if matches := r.FindSubmatch(line); len(matches) == 0 {
			t.Errorf("The line %q did not contain the regex, %q", line, r)
		}
		return nil
	}
}

func withTests(testFns ...testLineOutput) testIO {
	tt := testReadWriter{
		checkOutput: testFns,
	}

	return func(t *testing.T) *testReadWriter {
		tt.t = t
		return &tt
	}
}

type testIO func(*testing.T) *testReadWriter

func Test_Commands_inSeparateContainers(t *testing.T) {
	toRun := []struct {
		Name    string
		Host    string
		Cmd     tc.Executable
		IO      testIO
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
			IO: withTests(endOK),
		},
		{
			Name: "reload",
			Host: service.ID.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"reload"},
				User: service.ID.String(),
				Key:  defaultPrivateKey,
			},
			IO: withTests(endOK),
		},
		{
			Name: "maintenance",
			Host: service.ID.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"maintenance"},
				User: service.ID.String(),
				Key:  defaultPrivateKey,
			},
			IO: withTests(endOK),
		},
		{
			Name: "pub actor add",
			Host: service.ID.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"pub", "actor", "add", "--type", "Person", "--key-type", "RSA", "--tag", "#sysop", "jdoe"},
				User: service.ID.String(),
				Key:  defaultPrivateKey,
			},
			IO: withTests(
				withInput(matchesRegexp(passRegexp), "asd"),
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
			IO: withTests(
				withInput(matchesRegexp(passRegexp), "asd"),
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
			IO: withTests(endOK),
		},
		{
			Name: "password change",
			Host: service.ID.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"accounts", "pass", service.ID.String()},
				User: service.ID.String(),
				Key:  defaultPrivateKey,
			},
			IO: withTests(
				withInput(matchesRegexp(passRegexp), "asd"),
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
			IO: withTests(endOK),
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

			ctx, cancelFn := context.WithTimeout(context.Background(), 30*time.Second)
			t.Cleanup(cancelFn)

			cont, err := c.Init(ctx, t, images...)
			if err != nil {
				t.Fatalf("unable to initialize containers: %s", err)
			}

			t.Cleanup(func() {
				cont.Cleanup(t)
			})
			_, err = cont.RunCommand(ctx, test.Host, test.Cmd, test.IO(t))
			if !eqErrs(test.WantErr, err) {
				if test.Cmd == nil {
					t.Fatalf("Err received executing nil command %s: %+v", test.Host, diffErrs(test.WantErr, err))
				}
				t.Fatalf("Err received executing command %s->%v: %+v", test.Host, test.Cmd.AsCommand(), diffErrs(test.WantErr, err))
			}
		})
	}
}
