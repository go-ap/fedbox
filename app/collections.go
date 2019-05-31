package app

import (
	"fmt"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/handlers"
	"github.com/go-ap/storage"
	"net/http"
)

func reqURL(r *http.Request) string {
	scheme := "http"
	if Config.Secure || r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL.Path)
}

// HandleCollection serves content from the generic collection end-points
// that return ActivityPub objects or activities
func HandleCollection(typ handlers.CollectionType, r *http.Request, repo storage.CollectionLoader) (as.CollectionInterface, error) {
	var items as.CollectionInterface
	var err error

	ff, _ := activitypub.FromRequest(r)
	f, _ := ff.(*activitypub.Filters)
	f.ItemKey = append(f.ItemKey, activitypub.Hash(reqURL(r)))

	if !activitypub.ValidActivityCollection(string(typ)) {
		return nil, NotFoundf("collection '%s' not found", f.Collection)
	}

	items, _, err = repo.LoadCollection(f)
	if err != nil {
		return nil, err
	}

	return items, nil
}
