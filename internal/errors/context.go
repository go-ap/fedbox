package errors

import (
	"fmt"
	"github.com/go-ap/jsonld"
	_ "golang.org/x/xerrors"
)

// TODO(marius): get a proper context from
func context() jsonld.Context {
	return jsonld.Context{
		jsonld.ContextElement{
			Term: jsonld.Term("errors"),
			IRI: "/ns#errors",
		},
	}
}

// Render outputs the json encoded errors, with the JsonLD context for current
func Render(errs ...error) ([]byte, error) {
	errMap := make([]string, len(errs))
	for i, err := range errs {
		if false {
			// TODO(marius): this is for dev environment
			errMap[i] = fmt.Sprintf("%s: %w", err, err)
		} else {
			errMap[i] = fmt.Sprintf("%s", err)
		}
	}
	return jsonld.WithContext(context()).Marshal(struct{Errors []string `jsonld:"errors"`}{Errors: errMap})
}
