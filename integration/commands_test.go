package integration

import (
	"bufio"
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

type testOutput func(*testing.T, io.Reader)
type testLineOutput func(*testing.T, *bufio.Scanner)

func consumeAllLines(lineChecks ...testLineOutput) testOutput {
	return func(t *testing.T, reader io.Reader) {
		scanner := bufio.NewScanner(reader)
		scanner.Split(bufio.ScanLines)
		for _, lineCheck := range lineChecks {
			lineCheck(t, scanner)
		}
		if scanner.Scan() {
			t.Errorf("Not all output was consumed: %q", scanner.Bytes())
		}
	}
}

func anyOutput(t *testing.T, lr *bufio.Scanner) {
	if !lr.Scan() {
		t.Fatalf("Unable to read next line")
	}
}
func endOK(t *testing.T, lr *bufio.Scanner) {
	if !lr.Scan() {
		t.Fatalf("Unable to read next line")
	}
	last := lr.Bytes()
	ok := []byte("FedBOX SSH: OK")
	if !bytes.Equal(last, ok) {
		t.Errorf("Output line %q, expected: %q", last, ok)
	}
}

var urlRegexp = regexp.MustCompile(`(http|https://[a-zA-Z0-9./-]+)`)

func matchesRegexp(r *regexp.Regexp) testLineOutput {
	return func(t *testing.T, lr *bufio.Scanner) {
		if !lr.Scan() {
			t.Fatalf("Unable to read next line")
		}
		line := lr.Bytes()
		if matches := r.FindSubmatch(line); len(matches) > 1 {
			return
		}
		t.Errorf("The line %q did not contain the regex, %q", line, r)
	}
}

func Test_Commands_inSeparateContainers(t *testing.T) {
	toRun := []struct {
		Name    string
		Host    string
		Cmd     tc.Executable
		Output  testOutput
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
			Output: consumeAllLines(endOK),
		},
		{
			Name: "reload",
			Host: service.ID.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"reload"},
				User: service.ID.String(),
				Key:  defaultPrivateKey,
			},
			Output: consumeAllLines(endOK),
		},
		{
			Name: "maintenance",
			Host: service.ID.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"maintenance"},
				User: service.ID.String(),
				Key:  defaultPrivateKey,
			},
			Output: consumeAllLines(endOK),
		},
		{
			Name: "pub actor add",
			Host: service.ID.String(),
			Cmd: c.SSHCmd{
				Cmd:   []string{"pub", "actor", "add", "--type", "Person", "--key-type", "RSA", "--tag", "#sysop", "jdoe"},
				User:  service.ID.String(),
				Key:   defaultPrivateKey,
				Input: bytes.NewReader([]byte("asd\nasd\n")),
			},
			Output: consumeAllLines(anyOutput, anyOutput, matchesRegexp(urlRegexp), endOK),
		},
		{
			Name: "oauth client add",
			Host: service.ID.String(),
			Cmd: c.SSHCmd{
				Cmd:   []string{"oauth", "client", "add", "--redirect-uri", "http://127.0.0.1"},
				User:  service.ID.String(),
				Key:   defaultPrivateKey,
				Input: bytes.NewReader([]byte("asd\nasd\n")),
			},
			Output: consumeAllLines(anyOutput, anyOutput, matchesRegexp(urlRegexp), endOK),
		},
		{
			Name: "storage bootstrap",
			Host: service.ID.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"storage", "bootstrap"},
				User: service.ID.String(),
				Key:  defaultPrivateKey,
			},
			Output: consumeAllLines(endOK),
		},
		{
			Name: "password change",
			Host: service.ID.String(),
			Cmd: c.SSHCmd{
				Cmd:   []string{"accounts", "pass", service.ID.String()},
				User:  service.ID.String(),
				Key:   defaultPrivateKey,
				Input: bytes.NewReader([]byte("test\ntest\n")),
			},
			Output: consumeAllLines(anyOutput, anyOutput, endOK),
		},
		{
			Name: "stop",
			Host: service.ID.String(),
			Cmd: c.SSHCmd{
				Cmd:  []string{"stop"},
				User: service.ID.String(),
				Key:  defaultPrivateKey,
			},
			Output: consumeAllLines(endOK),
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
			out, err := cont.RunCommand(ctx, test.Host, test.Cmd)
			if !eqErrs(test.WantErr, err) {
				if test.Cmd == nil {
					t.Fatalf("Err received executing nil command %s: %+v", test.Host, diffErrs(test.WantErr, err))
				}
				t.Fatalf("Err received executing command %s->%v: %+v", test.Host, test.Cmd.AsCommand(), diffErrs(test.WantErr, err))
			}
			if test.Output != nil && out == nil {
				t.Fatalf("No output from command when it was expected %s->%v", test.Host, test.Cmd.AsCommand())
			}

			test.Output(t, out)
		})
	}
}
