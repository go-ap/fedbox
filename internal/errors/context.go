package errors

import (
	"fmt"
	"github.com/go-ap/jsonld"
	_ "golang.org/x/xerrors"
	"net/http"
)

// TODO(marius): get a proper context from
func context(r *http.Request) jsonld.Context {
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

// Render outputs the json encoded errors, with the JsonLD context for current
func Render(r *http.Request, errs ...error) (int, []byte) {
	errMap := make([]Http, len(errs))
	var status int
	for i, err := range errs {
		h := HttpError(err)
		errMap[i] = h
		status = h.Code
	}
	var dat []byte
	var err error

	m := struct {
		Errors []Http `jsonld:"errors"`
	}{Errors: errMap}
	if dat, err = jsonld.WithContext(context(r)).Marshal(m); err != nil {
		return http.StatusInternalServerError, dat
	}
	return status, dat
}
