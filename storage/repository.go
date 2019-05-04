package storage

import (
	s "github.com/go-ap/activitypub/storage"
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/fedbox/internal/errors"
	"strings"
)

type Paginator = s.Paginator

type ActivityPubLoader interface {
	Load(f s.Filterable) (as.ItemCollection, error)
}
type ActivityPubActivityLoader interface {
	LoadActivities(f s.Filterable) (as.ItemCollection, error)
}
type ActivityPubActorLoader interface {
	LoadActors(f s.Filterable) (as.ItemCollection, error)
}
type ActivityPubObjectLoader interface {
	LoadObject(f s.Filterable) (as.ItemCollection, error)
}

type Loader struct{}

func (l Loader) Load(f s.Filterable) (as.ItemCollection, error) {
	return nil, errors.NotImplementedf("maybe this isn't needed")
}
func (l Loader) LoadActivities(f s.Filterable) (as.ItemCollection, error) {
	return LoadActivities(f)
}
func (l Loader) LoadActors(f s.Filterable) (as.ItemCollection, error) {
	return LoadActors(f)
}
func (l Loader) LoadObjects(f s.Filterable) (as.ItemCollection, error) {
	return LoadObjects(f)
}

func LoadActivities(f s.Filterable) (as.ItemCollection, error) {
	items := make(as.ItemCollection, 3)
	items[0] = as.CreateNew("1", as.ObjectNew("1"))
	items[1] = as.LikeNew("2", as.ObjectNew("1"))
	items[2] = as.UpdateNew("3", as.ObjectNew("1"))

	ret := make(as.ItemCollection, 0)
	iris := f.IRIs()
	if len(iris) > 0 {
		for _, key := range iris {
			for _, it := range items {
				if strings.Contains(string(*it.GetID()), key.String()) {
					ret = append(ret, it)
				}
			}
		}
	} else {
		ret = items
	}
	return ret, nil
}

func LoadObjects(f s.Filterable) (as.ItemCollection, error) {
	items := make(as.ItemCollection, 6)
	items[0] = as.ObjectNew(as.ArticleType)
	items[1] = as.ObjectNew(as.NoteType)
	items[2] = as.ObjectNew(as.TombstoneType)
	items[3] = as.ObjectNew(as.AudioType)
	items[4] = as.ObjectNew(as.VideoType)
	items[5] = as.LinkNew("6", as.MentionType)

	ret := make(as.ItemCollection, 0)
	iris := f.IRIs()
	if len(iris) > 0 {
		for _, key := range iris {
			for _, it := range items {
				if strings.Contains(string(*it.GetID()), key.String()) {
					ret = append(ret, it)
				}
			}
		}
	} else {
		ret = items
	}
	return ret, nil
}

func LoadActors(f s.Filterable) (as.ItemCollection, error) {
	items := make(as.ItemCollection, 4)
	items[0] = as.PersonNew("alice")
	items[1] = as.PersonNew("bob")
	items[2] = as.ServiceNew("self")
	items[3] = as.GroupNew("dev")

	ret := make(as.ItemCollection, 0)
	iris := f.IRIs()
	if len(iris) > 0 {
		for _, key := range iris {
			for _, it := range items {
				if strings.Contains(string(*it.GetID()), key.String()) {
					ret = append(ret, it)
				}
			}
		}
	} else {
		ret = items
	}

	return ret, nil
}
