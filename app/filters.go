package app

import (
	"fmt"
	"net/http"
	"regexp"

	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/auth"
	"github.com/go-ap/errors"
	ap "github.com/go-ap/fedbox/activitypub"
	"github.com/go-ap/handlers"
)

// LoadCollectionFilters uses specific logic for adding elements to the filters when loading
// collections from the database.
func LoadCollectionFilters(r *http.Request, f *ap.Filters) error {
	err := LoadItemFilters(r, f)

	pr, _ := regexp.Compile(fmt.Sprintf("/(%s|%s|%s)(?:/([a-f0-9-]+)/%s)?", ap.ActorsType, ap.ActivitiesType, ap.ObjectsType, f.Collection))
	matches := pr.FindSubmatch([]byte(r.URL.Path))
	if len(matches) < 3 {
		return errors.NotFoundf("%s collection not found", f.Collection)
	} else {
		topCol := handlers.CollectionType(matches[1])
		hash := matches[2]
		if hash != nil {
			// The filter's semantics are different based on the collection type and base collection type
			// Activities: (nothing)?
			// Actors:
			//   Inbox: items that have actor as Audience
			//   Outbox, Shares, Liked, Likes: items that have actor as AttributedTo
			//   Following: actors that the current actor has in the followers list
			//   Followers: actors that have the current To in their followers list
			// Objects:
			//   Replies: items that have current object as context
			switch topCol {
			case ap.ActorsType:
				// TODO(marius): this needs to be moved somewhere where it makes more sense
				if loader, ok := actorLoader(r.Context()); ok {
					ff := ap.Filters{}
					ff.ItemKey = []ap.Hash{ap.Hash(hash)}
					switch f.Collection {
					case handlers.Inbox:
						if act, _, err := loader.LoadActors(&ff); err == nil {
							f.To = act.First()
						}
					case handlers.Outbox:
						fallthrough
					case handlers.Shares:
						fallthrough
					case handlers.Liked:
						fallthrough
					case handlers.Likes:
						if act, _, err := loader.LoadActors(&ff); err == nil {
							f.Author = act
						}
					case handlers.Followers:
						// TODO(marius) ? ? ?
					case handlers.Following:
						// TODO(marius) ? ? ?
					}
				}
			case ap.ObjectsType:
				// TODO(marius): this needs to be moved somewhere where it makes more sense
				if loader, ok := objectLoader(r.Context()); ok {
					ff := ap.Filters{ItemKey: []ap.Hash{ap.Hash(f.IRI)}}
					if act, _, err := loader.LoadObjects(&ff); err == nil {
						f.Parent = act.First()
					}
				}
				if hash != nil {
					f.ItemKey = []ap.Hash{ap.Hash(hash)}
				}
			case ap.ActivitiesType:
				if hash != nil {
					f.Key = []ap.Hash{ap.Hash(hash)}
				}
			}
		}
	}

	return err
}

// LoadItemFilters uses specific logic for adding elements to the filters when loading
// single items from the database.
func LoadItemFilters(r *http.Request, f *ap.Filters) error {
	f.IRI = as.IRI(reqURL(r))

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
