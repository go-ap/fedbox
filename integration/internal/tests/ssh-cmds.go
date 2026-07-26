package tests

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/client/c2s"
	c "github.com/go-ap/fedbox/integration/internal/containers"
	"github.com/google/go-cmp/cmp"
	tc "github.com/testcontainers/testcontainers-go"
	gossh "golang.org/x/crypto/ssh"
)

type ioTest struct {
	t           testing.TB
	m           sync.RWMutex
	outputCount int
	input       []byte
	checkOutput []LineOutputTest
}

func (t *ioTest) Read(p []byte) (n int, err error) {
	t.m.RLock()
	defer t.m.RUnlock()

	if len(t.input) == 0 {
		return n, nil
	}

	input := append(t.input, CR...)
	n = copy(p, input)
	t.t.Logf("IN: %s", t.input)

	t.input = t.input[:0]

	return n, nil
}

var CRLF = []byte{'\r', '\n'}
var CR = []byte{'\n'}

func (t *ioTest) Write(p []byte) (n int, err error) {
	t.m.Lock()
	defer t.m.Unlock()

	if bytes.Equal(p, CRLF) || bytes.Equal(p, CR) {
		return len(p), nil
	}
	t.t.Logf("OUT: %s", bytes.Trim(p, string(CRLF)))
	if len(t.checkOutput) == 0 {
		t.t.Errorf("output was provided, but no handler was provided in the test setup")
		return len(p), nil
	}
	// NOTE(marius): we need that the checkOutput functions count match the number of output buffers
	if t.outputCount >= len(t.checkOutput) {
		t.t.Errorf("not enough output checker functions %d for how many lines of output we want: %d", len(t.checkOutput)+1, t.outputCount)
		return len(p), nil
	}

	checkFn := t.checkOutput[t.outputCount]
	t.input = checkFn(t.t, bytes.Trim(p, string(CRLF)))
	t.outputCount++

	return len(p), nil
}

var _ io.ReadWriter = new(ioTest)

type LineOutputTest func(testing.TB, []byte) []byte

func WithTests(testFns ...LineOutputTest) ioTestFn {
	tt := ioTest{
		checkOutput: testFns,
	}

	return func(t *testing.T) *ioTest {
		tt.t = t
		return &tt
	}
}

type ioTestFn func(*testing.T) *ioTest

type RunnableTest interface {
	Label() string
	// Run is one way to run the tests directly from a func(*testing.T)
	Run(context.Context, c.Running, *testing.T)
	// Fn is a way to pass the test fn directly to testing.T.Run()
	Fn(context.Context, c.Running) func(*testing.T)
}

type CommandTest struct {
	Name    string
	Host    string
	Cmd     tc.Executable
	IO      ioTestFn
	WantErr error
}

func (c CommandTest) Label() string {
	return c.Name
}

func (c CommandTest) Fn(ctx context.Context, mocks c.Running) func(t *testing.T) {
	return func(t *testing.T) {
		c.Run(ctx, mocks, t)
	}
}

func (c CommandTest) Run(ctx context.Context, containers c.Running, t *testing.T) {
	_, err := containers.RunCommand(ctx, c.Host, c.Cmd, c.IO(t))
	if !EqErrs(c.WantErr, err) && !errors.Is(err, new(gossh.ExitMissingError)) {
		if c.Cmd == nil {
			t.Errorf("Error received executing nil command %s: %+v", c.Host, DiffErrs(c.WantErr, err))
		}
		t.Errorf("Error received executing command %s %v: %+v", c.Host, c.Cmd.AsCommand(), DiffErrs(c.WantErr, err))
	}
}

var _ RunnableTest = CommandTest{}

var EquateWeakErrors = cmp.FilterValues(areErrors, cmp.Comparer(compareErrors))

func areErrors(x, y any) bool {
	_, ok1 := x.(error)
	_, ok2 := y.(error)
	return ok1 && ok2
}

func compareErrors(x, y any) bool {
	xe := x.(error)
	ye := y.(error)
	return errors.Is(xe, ye) || errors.Is(ye, xe) || xe.Error() == ye.Error()
}

func EqErrs(want, got error) bool {
	return cmp.Equal(want, got, EquateWeakErrors)
}

func DiffErrs(want, got error) string {
	return cmp.Diff(want, got, EquateWeakErrors)
}

func AnyOutput(t testing.TB, line []byte) []byte {
	t.Logf("read %q", bytes.Trim(line, string(CRLF)))
	return nil
}

func EndOK(t testing.TB, line []byte) []byte {
	ok := []byte("FedBOX SSH: OK")
	if !bytes.Equal(line, ok) {
		t.Errorf("Output line %q, expected: %q", line, ok)
	}
	return nil
}

func MatchToken(t testing.TB, i []byte) []byte {
	remainder, found := bytes.CutPrefix(i, []byte("Authorization: "))
	if !found {
		t.Errorf("Invalid output for oauth token command (missing Authorization prefix): %s", i)
	}
	remainder, found = bytes.CutPrefix(remainder, []byte("Bearer"))
	if !found {
		t.Errorf("Invalid output for oauth token command (missing Bearer token type): %s", i)
	}
	return nil
}

func GetToken(token *c2s.BearerSigner) LineOutputTest {
	return func(t testing.TB, i []byte) []byte {
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

var (
	URLRegexp    = regexp.MustCompile(`((?:http|https)://[a-zA-Z0-9./-]+)`)
	PassMatch    = MatchesString(`Password: `)
	ConfirmMatch = MatchesString(` Confirm: `)
)

func WithInput(r LineOutputTest, input string) LineOutputTest {
	return func(t testing.TB, line []byte) []byte {
		// NOTE(marius): ignore any input returned by previous test
		_ = r(t, line)
		return []byte(input)
	}
}

func ExtractActorIRI(iri *vocab.IRI) LineOutputTest {
	// NOTE(marius): this expects the output:
	// Added "TYPE" "PREFERRED_USERNAME"
	// \thttps://URL
	return func(t testing.TB, line []byte) []byte {
		matches := URLRegexp.FindSubmatch(line)
		if len(matches) == 0 {
			t.Errorf("The line %q did not contain the regex, %q", line, URLRegexp)
			return nil
		}
		u, err := url.Parse(string(bytes.TrimSpace(matches[0])))
		if err != nil {
			t.Errorf("The line %q did not contain a valid URL, %v", line, err)
			return nil
		}
		*iri = vocab.IRI(u.String())
		return nil
	}
}

func MatchesRegexp(r *regexp.Regexp) LineOutputTest {
	return func(t testing.TB, line []byte) []byte {
		matches := r.FindSubmatch(line)
		if len(matches) == 0 {
			t.Errorf("The line %q did not contain the regex, %q", line, r)
		}
		if _, err := url.Parse(string(bytes.TrimSpace(matches[0]))); err != nil {
			t.Errorf("The line %q did not contain a valid URL, %v", line, err)
		}
		return nil
	}
}

func MatchesString(s string) LineOutputTest {
	return func(t testing.TB, line []byte) []byte {
		if !bytes.Equal(line, []byte(s)) {
			t.Errorf("The line %q did not match expected, %q", line, s)
		}
		return nil
	}
}
