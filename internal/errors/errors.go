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
	return n.Err
}
func (n notValid) Unwrap() error {
	return n.Err
}
func (n notImplemented) Unwrap() error {
	return n.Err
}
func (n notSupported) Unwrap() error {
	return n.Err
}
func (b badRequest) Unwrap() error {
	return b.Err
}
func (t timeout) Unwrap() error {
	return t.Err
}
func (u unauthorized) Unwrap() error {
	return u.Err
}
func (m methodNotAllowed) Unwrap() error {
	return m.Err
}
func (f forbidden) Unwrap() error {
	return f.Err
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
