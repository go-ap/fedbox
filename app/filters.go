package app

import (
	"fmt"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/fedbox/activitypub"
	"net/http"
	"regexp"
)

func LoadToFilters(r *http.Request, f *activitypub.Filters) error {
	pr, _ := regexp.Compile(fmt.Sprintf("/(actors|items|activities)/(\\w+)/%s", f.Collection))
	matches := pr.FindSubmatch([]byte(r.URL.Path))
	if len(matches) < 3 {
		return NotFoundf("%s collection not found", f.Collection)
	} else {
		col := matches[1]
		url := reqURL(r)
		switch string(col) {
		case "actors":
			// TODO(marius): this needs to be moved somewhere where it makes more sense
			if loader, ok := Loader(r.Context()); ok {
				ff := activitypub.Filters{
					Type: []as.ActivityVocabularyType{
						as.PersonType,
						as.GroupType,
						as.ApplicationType,
						as.ServiceType,
						as.OrganizationType,
					},
					Key: []activitypub.Hash{activitypub.Hash(url)},
				}
				if act, _, err := loader.LoadActors(&ff); err == nil {
					f.Owner = act
				}
			}
		case "items":
			f.ItemKey = []activitypub.Hash{activitypub.Hash(url)}
		case "activities":
			f.Key = []activitypub.Hash{activitypub.Hash(url)}
		}
	}

	// TODO(marius): this needs to be moved somewhere where it makes more sense
	ctxVal := r.Context().Value(ActorKey)
	if a, ok := ctxVal.(as.Actor); ok {
		f.Actor = a
	}
	return nil
}
