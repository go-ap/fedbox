package app

import (
	"fmt"
	as "github.com/go-ap/activitystreams"
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
			i := as.IRI(fmt.Sprintf("%s%s", f.IRI, k))
			f.URL = append(f.URL, i)
		}
	}

	if auth, ok := auth.ActorContext(r.Context()); ok {
		f.Authenticated = &auth
	}
	return nil
}
