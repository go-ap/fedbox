package app

import (
	"fmt"
	"github.com/go-ap/activitypub/storage"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/fedbox/internal/context"
	st "github.com/go-ap/fedbox/storage"
	"net/http"
)

func reqURL(r *http.Request, path string) string {
	scheme := "http"
	if Config.Secure || r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s%s", scheme, r.Host, path)
}

// HandleCollection serves content from the generic collection end-points
// that return ActivityPub objects or activities
func HandleCollection(r *http.Request) (as.CollectionInterface, error) {
	var items as.CollectionInterface
	var err error

	f := &st.Filters{}
	f.FromRequest(r)
	f.ItemKey = append(f.ItemKey, st.Hash(reqURL(r, r.URL.Path)))

	if !st.ValidActivityCollection(string(f.Collection)) {
		return nil, NotFoundf("collection '%s' not found", f.Collection)
	}

	var repo storage.CollectionLoader
	var ok bool
	if repo, ok = context.CollectionLoader(r.Context()); !ok {
		return nil, NotValidf("unable to load storage")
	}
	items, _, err = repo.LoadCollection(f)
	if err != nil {
		return nil, err
	}

	return items, nil
}
