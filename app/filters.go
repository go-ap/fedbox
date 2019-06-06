package app

import (
	"fmt"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"net/http"
	"regexp"
)

func LoadCollectionFilters(r *http.Request, f *ap.Filters) error {
	err := LoadItemFilters(r, f)
	f.ItemKey = append(f.ItemKey, ap.Hash(reqURL(r)))
	return err
}

func LoadItemFilters(r *http.Request, f *ap.Filters) error {
	pr, _ := regexp.Compile(fmt.Sprintf("/(actors|items|activities)/(\\w+)/%s", f.Collection))
	matches := pr.FindSubmatch([]byte(r.URL.Path))
	if len(matches) < 3 {
		return errors.NotFoundf("%s collection not found", f.Collection)
	} else {
		col := matches[1]
		url := reqURL(r)
		switch string(col) {
		case "actors":
			// TODO(marius): this needs to be moved somewhere where it makes more sense
			if loader, ok := Loader(r.Context()); ok {
				ff := ap.Filters{
					Type: []as.ActivityVocabularyType{
						as.PersonType,
						as.GroupType,
						as.ApplicationType,
						as.ServiceType,
						as.OrganizationType,
					},
					Key: []ap.Hash{ap.Hash(url)},
				}
				if act, _, err := loader.LoadActors(&ff); err == nil {
					f.Owner = act
				}
			}
		case "items":
			f.ItemKey = []ap.Hash{ap.Hash(url)}
		case "activities":
			f.Key = []ap.Hash{ap.Hash(url)}
		}
	}

	// TODO(marius): this needs to be moved somewhere where it makes more sense
	ctxVal := r.Context().Value(ActorKey)
	if a, ok := ctxVal.(as.Actor); ok {
		f.Actor = a
	}
	return nil
}
