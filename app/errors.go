package app

import (
	"bytes"
	"fmt"
	"github.com/go-ap/fedbox/internal/errors"
	"github.com/go-ap/jsonld"
	xerr "golang.org/x/xerrors"
	"net/http"
	"strconv"
)

type notFound struct {
	errors.Err
}

type methodNotAllowed struct {
	errors.Err
}

type notValid struct {
	errors.Err
}

type forbidden struct {
	errors.Err
}

type notImplemented struct {
	errors.Err
}

type badRequest struct {
	errors.Err
}

type unauthorized struct {
	errors.Err
	challenge string
}

type notSupported struct {
	errors.Err
}

type timeout struct {
	errors.Err
}

func wrapErr(err error, s string, args ...interface{}) errors.Err {
	e := errors.Annotatef(err, s, args...)
	asErr := errors.Err{}
	xerr.As(e, &asErr)
	return asErr
}

func NotFoundf(s string, args ...interface{}) error {
	return &notFound{wrapErr(nil, s, args...)}
}

func NewNotFound(e error, s string, args ...interface{}) error {
	return &notFound{wrapErr(e, s, args...)}
}

func MethodNotAllowedf(s string, args ...interface{}) error {
	return &methodNotAllowed{wrapErr(nil, s, args...)}
}

func NewMethodNotAllowed(e error, s string, args ...interface{}) error {
	return &methodNotAllowed{wrapErr(e, s, args...)}
}

func NotValidf(s string, args ...interface{}) error {
	return &notValid{wrapErr(nil, s, args...)}
}

func NewNotValid(e error, s string, args ...interface{}) error {
	return &notValid{wrapErr(e, s, args...)}
}

func Forbiddenf(s string, args ...interface{}) error {
	return &forbidden{wrapErr(nil, s, args...)}
}

func NewForbidden(e error, s string, args ...interface{}) error {
	return &forbidden{wrapErr(e, s, args...)}
}

func NotImplementedf(s string, args ...interface{}) error {
	return &notImplemented{wrapErr(nil, s, args...)}
}

func NewNotImplemented(e error, s string, args ...interface{}) error {
	return &notImplemented{wrapErr(e, s, args...)}
}

func BadRequestf(s string, args ...interface{}) error {
	return &badRequest{wrapErr(nil, s, args...)}
}
func NewBadRequest(e error, s string, args ...interface{}) error {
	return &badRequest{wrapErr(e, s, args...)}
}
func Unauthorizedf(s string, args ...interface{}) error {
	return &unauthorized{Err: wrapErr(nil, s, args...)}
}
func NewUnauthorized(e error, s string, args ...interface{}) error {
	return &unauthorized{Err: wrapErr(e, s, args...)}
}
func NotSupportedf(s string, args ...interface{}) error {
	return &notSupported{wrapErr(nil, s, args...)}
}
func NewNotSupported(e error, s string, args ...interface{}) error {
	return &notSupported{wrapErr(e, s, args...)}
}
func NotTimeoutf(s string, args ...interface{}) error {
	return &timeout{wrapErr(nil, s, args...)}
}
func NewTimeout(e error, s string, args ...interface{}) error {
	return &timeout{wrapErr(e, s, args...)}
}
func IsBadRequest(e error) bool {
	_, ok := e.(*badRequest)
	return ok
}
func IsForbidden(e error) bool {
	_, okp := e.(*forbidden)
	_, oks := e.(forbidden)
	return okp || oks
}
func IsNotSupported(e error) bool {
	_, okp := e.(*notSupported)
	_, oks := e.(notSupported)
	return okp || oks
}
func IsMethodNotAllowed(e error) bool {
	_, okp := e.(*methodNotAllowed)
	_, oks := e.(methodNotAllowed)
	return okp || oks
}
func IsNotFound(e error) bool {
	_, okp := e.(*notFound)
	_, oks := e.(notFound)
	return okp || oks
}
func IsNotImplemented(e error) bool {
	_, okp := e.(*notImplemented)
	_, oks := e.(notImplemented)
	return okp || oks
}
func IsUnauthorized(e error) bool {
	_, okp := e.(*unauthorized)
	_, oks := e.(unauthorized)
	return okp || oks
}
func IsTimeout(e error) bool {
	_, okp := e.(*timeout)
	_, oks := e.(timeout)
	return okp || oks
}
func IsNotValid(e error) bool {
	_, okp := e.(*notValid)
	_, oks := e.(notValid)
	return okp || oks
}
func (n notFound) Is(e error) bool {
	return IsNotFound(e)
}
func (n notValid) Is(e error) bool {
	return IsNotValid(e)
}
func (n notImplemented) Is(e error) bool {
	return IsNotImplemented(e)
}
func (n notSupported) Is(e error) bool {
	return IsNotSupported(e)
}
func (b badRequest) Is(e error) bool {
	return IsBadRequest(e)
}
func (t timeout) Is(e error) bool {
	return IsTimeout(e)
}
func (u unauthorized) Is(e error) bool {
	return IsUnauthorized(e)
}
func (m methodNotAllowed) Is(e error) bool {
	return IsMethodNotAllowed(m)
}
func (f forbidden) Is(e error) bool {
	return IsForbidden(e)
}
func (n notFound) Unwrap() error {
	return n.Err.Unwrap()
}
func (n notValid) Unwrap() error {
	return n.Err.Unwrap()
}
func (n notImplemented) Unwrap() error {
	return n.Err.Unwrap()
}
func (n notSupported) Unwrap() error {
	return n.Err.Unwrap()
}
func (b badRequest) Unwrap() error {
	return b.Err.Unwrap()
}
func (t timeout) Unwrap() error {
	return t.Err.Unwrap()
}
func (u unauthorized) Unwrap() error {
	return u.Err.Unwrap()
}
func (m methodNotAllowed) Unwrap() error {
	return m.Err.Unwrap()
}
func (f forbidden) Unwrap() error {
	return f.Err.Unwrap()
}
func (n *notFound) As(err interface{}) bool {
	switch x := err.(type) {
	case **notFound:
		*x = n
	case *errors.Err:
		return n.Err.As(x)
	default:
		return false
	}
	return true
}
func (n *notValid) As(err interface{}) bool {
	switch x := err.(type) {
	case **notValid:
		*x = n
	case *errors.Err:
		return n.Err.As(x)
	default:
		return false
	}
	return true
}
func (n *notImplemented) As(err interface{}) bool {
	switch x := err.(type) {
	case **notImplemented:
		*x = n
	case *errors.Err:
		return n.Err.As(x)
	default:
		return false
	}
	return true
}
func (n *notSupported) As(err interface{}) bool {
	switch x := err.(type) {
	case **notSupported:
		*x = n
	case *errors.Err:
		return n.Err.As(x)
	default:
		return false
	}
	return true
}
func (b *badRequest) As(err interface{}) bool {
	switch x := err.(type) {
	case **badRequest:
		*x = b
	case *errors.Err:
		return b.Err.As(x)
	default:
		return false
	}
	return true
}
func (t *timeout) As(err interface{}) bool {
	switch x := err.(type) {
	case **timeout:
		*x = t
	case *errors.Err:
		return t.Err.As(x)
	default:
		return false
	}
	return true
}
func (u *unauthorized) As(err interface{}) bool {
	switch x := err.(type) {
	case **unauthorized:
		*x = u
	case *errors.Err:
		return u.Err.As(x)
	default:
		return false
	}
	return true
}
func (m *methodNotAllowed) As(err interface{}) bool {
	switch x := err.(type) {
	case **methodNotAllowed:
		*x = m
	case *errors.Err:
		return m.Err.As(x)
	default:
		return false
	}
	return true
}
func (f *forbidden) As(err interface{}) bool {
	switch x := err.(type) {
	case **forbidden:
		*x = f
	case *errors.Err:
		return f.Err.As(x)
	default:
		return false
	}
	return true
}

func NewUnauthorizedWithChallenge(c string, e error, s string, args ...interface{}) error {
	return &unauthorized{Err: wrapErr(e, s, args...), challenge: c}
}

func Challenge(err error) string {
	un := unauthorized{}
	if ok := xerr.As(err, &un); ok {
		return un.challenge
	}
	return ""
}

// ErrorHandlerFn
type ErrorHandlerFn func(http.ResponseWriter, *http.Request) error

// ServeHTTP implements the http.Handler interface for the ItemHandlerFn type
func (h ErrorHandlerFn) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var dat []byte
	var status int

	if err := h(w, r); err != nil {
		status, dat = RenderErrors(r, err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(dat)
}

func HandleError(e error) ErrorHandlerFn {
	return func(w http.ResponseWriter, r *http.Request) error {
		return e
	}
}

type Http struct {
	Code     int    `jsonld:"status"`
	Message  string `jsonld:"message"`
	Trace    *Stack `jsonld:"trace,omitempty"`
	Location string `jsonld:"location,omitempty"`
}

func HttpErrors(err error) (int, []Http) {
	https := make([]Http, 0)

	load := func(err error) Http {
		var loc string
		var trace *Stack
		var msg string
		switch e := err.(type) {
		case *errors.Err:
			msg = fmt.Sprintf("%s", e.Error())
			if errors.IncludeBacktrace {
				trace, _ = parseStack(e.StackTrace())
				f, l := e.Location()
				if len(f) > 0 {
					loc = fmt.Sprintf("%s:%d", f, l)
				}
			}
		default:
			local := new(errors.Err)
			if ok := xerr.As(err, local); ok {
				if errors.IncludeBacktrace {
					trace, _ = parseStack(local.StackTrace())
					f, l := local.Location()
					if len(f) > 0 {
						loc = fmt.Sprintf("%s:%d", f, l)
					}
				}
			}
			msg = err.Error()
		}

		return Http{
			Message:  msg,
			Trace:    trace,
			Location: loc,
			Code:     httpErrorResponse(err),
		}
	}
	code := httpErrorResponse(err)
	https = append(https, load(err))
	for {
		if err = xerr.Unwrap(err); err != nil {
			https = append(https, load(err))
		} else {
			break
		}
	}

	return code, https
}

func httpErrorResponse(e error) int {
	if IsBadRequest(e) {
		return http.StatusBadRequest
	}
	if IsUnauthorized(e) {
		return http.StatusUnauthorized
	}
	// http.StatusPaymentRequired
	if IsForbidden(e) {
		return http.StatusForbidden
	}
	if IsNotFound(e) {
		return http.StatusNotFound
	}
	if IsMethodNotAllowed(e) {
		return http.StatusMethodNotAllowed
	}
	if IsNotValid(e) {
		return http.StatusNotAcceptable
	}
	//  http.StatusProxyAuthRequired
	//  http.StatusRequestTimeout
	//  TODO(marius): http.StatusConflict
	//  TODO(marius): http.StatusGone
	//  http.StatusLengthRequres
	//  http.StatusPreconditionFailed
	//  http.StatusRequestEntityTooLarge
	//  http.StatusRequestURITooLong
	//  TODO(marius): http.StatusUnsupportedMediaType
	//  http.StatusRequestedRangeNotSatisfiable
	//  http.StatusExpectationFailed
	//  http.StatusTeapot
	//  http.StatusMisdirectedRequest
	//  http.StatusUnprocessableEntity
	//  http.StatusLocked
	//  http.StatusFailedDependency
	//  http.StatusTooEarly
	//  http.StatusTooManyRequests
	//  http.StatusRequestHeaderFieldsTooLarge
	//  http.StatusUnavailableForLegalReason

	//  http.StatusInternalServerError
	//  http.StatusInternalServerError
	if IsNotImplemented(e) {
		return http.StatusNotImplemented
	}
	if IsNotSupported(e) {
		return http.StatusHTTPVersionNotSupported
	}

	if IsTimeout(e) {
		return http.StatusGatewayTimeout
	}

	return http.StatusInternalServerError
}

// StackFunc is a function call in the backtrace
type StackFunc struct {
	Name    string  `json:"name"`
	ArgPtrs []int64 `json:"name,omitempty"`
}

// StackElement represents a stack call including file, line, and function call
type StackElement struct {
	File   string `jsonld:"file"`
	Line   int64  `jsonld:"line"`
	Callee string `jsonld:"calee,omitempty"`
	Addr   int64  `jsonld:"address,omitempty"`
}

// Stack is an array of stack elements representing the parsed relevant bits of a backtrace
// Relevant in this ctxt means, it strips the calls that are happening in the package
type Stack []StackElement

func parseStack(b []byte) (*Stack, error) {
	lvl := 2 // go up the stack call tree to hide the two internal calls
	lines := bytes.Split(b, []byte("\n"))

	if len(lines) <= lvl*2+1 {
		return nil, errors.Newf("invalid stack trace")
	}

	skipLines := lvl * 2
	stackLen := (len(lines) - 1 - skipLines) / 2
	relLines := lines[1+skipLines:]

	stack := make(Stack, stackLen)
	for i, curLine := range relLines {
		cur := i / 2
		if len(curLine) == 0 {
			continue
		}
		curStack := stack[cur]
		if i%2 == 0 {
			// function line
			curStack.Callee = string(curLine)
			//elems := bytes.Split(curLine, []byte("("))
			//curStack.Callee.Name = string(elems[0])
			//argsLine := bytes.Trim(elems[1], ")")
			//args := bytes.Split(argsLine, []byte(","))
			//curStack.Callee.ArgPtrs = make([]int64, len(args))
			//for j, arg := range args {
			//	curStack.Callee.ArgPtrs[j], _ = strconv.ParseInt(string(bytes.Trim(arg, " ")), 16, 64)
			//}
		} else {
			// file line
			curLine = bytes.Trim(curLine, "\t")
			elems := bytes.Split(curLine, []byte(":"))
			curStack.File = string(elems[0])

			elems1 := bytes.Split(elems[1], []byte(" "))
			cnt := len(elems1)
			if cnt > 0 {
				curStack.Line, _ = strconv.ParseInt(string(elems1[0]), 10, 64)
			}
			if cnt > 1 {
				curStack.Addr, _ = strconv.ParseInt(string(elems1[1]), 16, 64)
			}
		}
		stack[cur] = curStack
	}
	return &stack, nil
}

// TODO(marius): get a proper ctxt from
func ctxt(r *http.Request) jsonld.Context {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return jsonld.Context{
		jsonld.ContextElement{
			Term: jsonld.Term("errors"),
			IRI:  jsonld.IRI(fmt.Sprintf("%s://%s/ns#errors", scheme, r.Host)),
		},
	}
}

// RenderErrors outputs the json encoded errors, with the JsonLD ctxt for current
func RenderErrors(r *http.Request, errs ...error) (int, []byte) {
	errMap := make([]Http, 0)
	var status int
	for _, err := range errs {
		code, more := HttpErrors(err)
		errMap = append(errMap, more...)
		status = code
	}
	var dat []byte
	var err error

	m := struct {
		Errors []Http `jsonld:"errors"`
	}{Errors: errMap}
	if dat, err = jsonld.WithContext(ctxt(r)).Marshal(m); err != nil {
		return http.StatusInternalServerError, dat
	}
	return status, dat
}
