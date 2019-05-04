package app

import (
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/fedbox/internal/errors"
)

func LoadActivities(f Filters) (as.ItemCollection, error) {
	items := make(as.ItemCollection, 3)
	items[0] = as.CreateNew("1", as.ObjectNew("1"))
	items[1] = as.LikeNew("2", as.ObjectNew("1"))
	items[2] = as.UpdateNew("3", as.ObjectNew("1"))

	return items, nil

	return nil, errors.NotImplementedf("loading activities")
}

func LoadObjects(f Filters) (as.ItemCollection, error) {
	return nil, errors.NotImplementedf("loading objects")
}

func LoadActors(f Filters) (as.ItemCollection, error) {
	return nil, errors.NotImplementedf("loading actors")
}
