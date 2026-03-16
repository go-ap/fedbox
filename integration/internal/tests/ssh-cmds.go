package tests

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"testing"

	c "github.com/go-ap/fedbox/integration/internal/containers"
	"github.com/google/go-cmp/cmp"
	tc "github.com/testcontainers/testcontainers-go"
	gossh "golang.org/x/crypto/ssh"
)

type ioTest struct {
	t           *testing.T
	m           sync.RWMutex
	lineCount   int
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

	checker := t.checkOutput[t.lineCount]
	t.input = checker(t.t, bytes.Trim(p, string(CRLF)))
	t.lineCount++

	return len(p), nil
}

var _ io.ReadWriter = new(ioTest)

type LineOutputTest func(*testing.T, []byte) []byte

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

func (c CommandTest) Run(ctx context.Context, mocks c.Running, t *testing.T) {
	_, err := mocks.RunCommand(ctx, c.Host, c.Cmd, c.IO(t))
	if !eqErrs(c.WantErr, err) && !errors.Is(err, new(gossh.ExitMissingError)) {
		if c.Cmd == nil {
			t.Fatalf("Err received executing nil command %s: %+v", c.Host, diffErrs(c.WantErr, err))
		}
		t.Fatalf("Err received executing command %s->%v: %+v", c.Host, c.Cmd.AsCommand(), diffErrs(c.WantErr, err))
	}
}

var _ RunnableTest = CommandTest{}

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
