package errors

import (
	"fmt"
	xerr "golang.org/x/xerrors"
	"reflect"
	"runtime"
	"runtime/debug"
)

type notFound struct {
	Err
}

type methodNotAllowed struct {
	Err
}

type notValid struct {
	Err
}

type forbidden struct {
	Err
}

type notImplemented struct {
	Err
}

type badRequest struct {
	Err
}

type unauthorized struct {
	Err
	challenge string
}

type notSupported struct {
	Err
}

type timeout struct {
	Err
}

type Err struct {
	c error
	m string
	t []byte
	l int64
	f string
}

func (e Err) Error() string {
	return e.m
}

type wrapper interface {
	Unwrap() error
}

func (e Err) Unwrap() error {
	return e.c
}

func Details(e error) error {
	if x, ok := e.(wrapper); ok {
		return x.Unwrap()
	}
	return nil
}

func (e *Err) Location() (string, int64) {
	return e.f, e.l
}

func (e *Err) StackTrace() string {
	return string(e.t)
}

func Annotate(e error, s string) error {
	return Annotatef(e, s)
}

func Annotatef(e error, s string, args ...interface{}) error {
	_, file, line, _ := runtime.Caller(1)
	return &Err{
		c: e,
		m: fmt.Sprintf(s, args...),
		t: debug.Stack(),
		f: file,
		l: int64(line),
	}
}

func New(s string) error {
	_, file, line, _ := runtime.Caller(1)
	return &Err{
		m: s,
		t: debug.Stack(),
		f: file,
		l: int64(line),
	}
}

func wrap(e error, s string, args ...interface{}) Err {
	_, file, line, _ := runtime.Caller(2)
	return Err{
		c: e,
		m: fmt.Sprintf(s, args...),
		t: debug.Stack(),
		f: file,
		l: int64(line),
	}
}

func Errorf(s string, args ...interface{}) error {
	return &Err{
		m: fmt.Sprintf(s, args...),
		t: debug.Stack(),
	}
}

func NotFoundf(s string, args ...interface{}) error {
	return &notFound{wrap(nil, s, args...)}
}

func NewNotFound(e error, s string, args ...interface{}) error {
	return &notFound{wrap(e, s, args...)}
}

func MethodNotAllowedf(s string, args ...interface{}) error {
	return &methodNotAllowed{wrap(nil, s, args...)}
}

func NewMethodNotAllowed(e error, s string, args ...interface{}) error {
	return &methodNotAllowed{wrap(e, s, args...)}
}

func NotValidf(s string, args ...interface{}) error {
	return &notValid{wrap(nil, s, args...)}
}

func NewNotValid(e error, s string, args ...interface{}) error {
	return &notValid{wrap(e, s, args...)}
}

func Forbiddenf(s string, args ...interface{}) error {
	return &forbidden{wrap(nil, s, args...)}
}

func NewForbidden(e error, s string, args ...interface{}) error {
	return &forbidden{wrap(e, s, args...)}
}

func NotImplementedf(s string, args ...interface{}) error {
	return &notImplemented{wrap(nil, s, args...)}
}

func NewNotImplemented(e error, s string, args ...interface{}) error {
	return &notImplemented{wrap(e, s, args...)}
}

func BadRequestf(s string, args ...interface{}) error {
	return &badRequest{wrap(nil, s, args...)}
}
func NewBadRequest(e error, s string, args ...interface{}) error {
	return &badRequest{wrap(e, s, args...)}
}
func Unauthorizedf(s string, args ...interface{}) error {
	return &unauthorized{Err: wrap(nil, s, args...)}
}
func NewUnauthorized(e error, s string, args ...interface{}) error {
	return &unauthorized{Err: wrap(e, s, args...)}
}
func NotSupportedf(s string, args ...interface{}) error {
	return &notSupported{wrap(nil, s, args...)}
}
func NewNotSupported(e error, s string, args ...interface{}) error {
	return &notSupported{wrap(e, s, args...)}
}
func NotTimeoutf(s string, args ...interface{}) error {
	return &timeout{wrap(nil, s, args...)}
}
func NewTimeout(e error, s string, args ...interface{}) error {
	return &timeout{wrap(e, s, args...)}
}
func IsBadRequest(e error) bool {
	return xerr.Is(e, badRequest{})
}
func IsForbidden(e error) bool {
	return xerr.Is(e, forbidden{})
}
func IsNotSupported(e error) bool {
	return xerr.Is(e, notSupported{})
}
func IsMethodNotAllowed(e error) bool {
	return xerr.Is(e, methodNotAllowed{})
}
func IsNotFound(e error) bool {
	return xerr.Is(e, notFound{})
}
func IsNotImplemented(e error) bool {
	return xerr.Is(e, notImplemented{})
}
func IsUnauthorized(e error) bool {
	return xerr.Is(e, unauthorized{})
}
func IsTimeout(e error) bool {
	return xerr.Is(e, timeout{})
}
func IsNotValid(e error) bool {
	return xerr.Is(e, notValid{})
}
func isA(err1, err2 error) bool {
	return reflect.TypeOf(err1) == reflect.TypeOf(err2)
}
func (n notFound) Is(e error) bool {
	return isA(n, e) || isA(n.Err, e)
}
func (n notValid) Is(e error) bool {
	return isA(n, e) || isA(n.Err, e)
}
func (n notImplemented) Is(e error) bool {
	return isA(n, e) || isA(n.Err, e)
}
func (n notSupported) Is(e error) bool {
	return isA(n, e) || isA(n.Err, e)
}
func (b badRequest) Is(e error) bool {
	return isA(b, e) || isA(b.Err, e)
}
func (t timeout) Is(e error) bool {
	return isA(t, e) || isA(t.Err, e)
}
func (u unauthorized) Is(e error) bool {
	return isA(u, e) || isA(u.Err, e)
}
func (m methodNotAllowed) Is(e error) bool {
	return isA(m, e) || isA(m.Err, e)
}
func (f forbidden) Is(e error) bool {
	return isA(f, e) || isA(f.Err, e)
}
func (n notFound) Unwrap() error {
	return n.Err.c
}
func (n notValid) Unwrap() error {
	return n.Err.c
}
func (n notImplemented) Unwrap() error {
	return n.Err.c
}
func (n notSupported) Unwrap() error {
	return n.Err.c
}
func (b badRequest) Unwrap() error {
	return b.Err.c
}
func (t timeout) Unwrap() error {
	return t.Err.c
}
func (u unauthorized) Unwrap() error {
	return u.Err.c
}
func (m methodNotAllowed) Unwrap() error {
	return m.Err.c
}
func (f forbidden) Unwrap() error {
	return f.Err.c
}
func (n *notFound) As(err interface{}) bool {
	switch x := err.(type) {
	case **notFound:
		*x = n
	case *Err:
		*x = Err{
			c: n.Err.c,
			m: n.Err.m,
			t: n.Err.t,
			l: n.Err.l,
			f: n.Err.f,
		}
	default:
		return false
	}
	return true
}
func (n *notValid) As(err interface{}) bool {
	switch x := err.(type) {
	case **notValid:
		*x = n
	case *Err:
		*x = Err{
			c: n.Err.c,
			m: n.Err.m,
			t: n.Err.t,
			l: n.Err.l,
			f: n.Err.f,
		}
	default:
		return false
	}
	return true
}
func (n *notImplemented) As(err interface{}) bool {
	switch x := err.(type) {
	case **notImplemented:
		*x = n
	case *Err:
		*x = Err{
			c: n.Err.c,
			m: n.Err.m,
			t: n.Err.t,
			l: n.Err.l,
			f: n.Err.f,
		}
	default:
		return false
	}
	return true
}
func (n *notSupported) As(err interface{}) bool {
	switch x := err.(type) {
	case **notSupported:
		*x = n
	case *Err:
		*x = Err{
			c: n.Err.c,
			m: n.Err.m,
			t: n.Err.t,
			l: n.Err.l,
			f: n.Err.f,
		}
	default:
		return false
	}
	return true
}
func (b *badRequest) As(err interface{}) bool {
	switch x := err.(type) {
	case **badRequest:
		*x = b
	case *Err:
		*x = Err{
			c: b.Err.c,
			m: b.Err.m,
			t: b.Err.t,
			l: b.Err.l,
			f: b.Err.f,
		}
	default:
		return false
	}
	return true
}
func (t *timeout) As(err interface{}) bool {
	switch x := err.(type) {
	case **timeout:
		*x = t
	case *Err:
		*x = Err{
			c: t.Err.c,
			m: t.Err.m,
			t: t.Err.t,
			l: t.Err.l,
			f: t.Err.f,
		}
	default:
		return false
	}
	return true
}
func (u *unauthorized) As(err interface{}) bool {
	switch x := err.(type) {
	case **unauthorized:
		*x = u
	case *Err:
		*x = Err{
			c: u.Err.c,
			m: u.Err.m,
			t: u.Err.t,
			l: u.Err.l,
			f: u.Err.f,
		}
	default:
		return false
	}
	return true
}
func (m *methodNotAllowed) As(err interface{}) bool {
	switch x := err.(type) {
	case **methodNotAllowed:
		*x = m
	case *Err:
		*x = Err{
			c: m.Err.c,
			m: m.Err.m,
			t: m.Err.t,
			l: m.Err.l,
			f: m.Err.f,
		}
	default:
		return false
	}
	return true
}
func (f *forbidden) As(err interface{}) bool {
	switch x := err.(type) {
	case **forbidden:
		*x = f
	case *Err:
		*x = Err{
			c: f.Err.c,
			m: f.Err.m,
			t: f.Err.t,
			l: f.Err.l,
			f: f.Err.f,
		}
	default:
		return false
	}
	return true
}

func NewUnauthorizedWithChallenge(c string, e error, s string, args ...interface{}) error {
	return &unauthorized{Err: wrap(e, s, args...), challenge: c}
}
func Challenge(err error) string {
	un := unauthorized{}
	if ok := xerr.As(err, &un); ok {
		return un.challenge
	}
	return ""
}
