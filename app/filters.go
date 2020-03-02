package app

import (
	"fmt"
	"github.com/go-ap/auth"
	ap "github.com/go-ap/fedbox/activitypub"
	"net/http"
)

// LoadCollectionFilters uses specific logic for adding elements to the filters when loading
// collections from the database.
func LoadCollectionFilters(r *http.Request, f *ap.Filters) error {
	return LoadItemFilters(r, f)
}

// LoadItemFilters uses specific logic for adding elements to the filters when loading
// single items from the database.
func LoadItemFilters(r *http.Request, f *ap.Filters) error {
	if len(f.Key) != 0 {
		for _, k := range f.Key {
			i := ap.CompStr{Str: fmt.Sprintf("%s%s", f.IRI, k)}
			f.URL = append(f.URL, i)
		}
	}

	if auth, ok := auth.ActorContext(r.Context()); ok {
		f.Authenticated = &auth
		if f.Object != nil {
			f.Object.Authenticated = f.Authenticated
		}
		if f.Actor != nil {
			f.Actor.Authenticated = f.Authenticated
		}
		if f.Target != nil {
			f.Target.Authenticated = f.Authenticated
		}
	}

	return nil
}
