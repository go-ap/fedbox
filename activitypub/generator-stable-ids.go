//go:build integration

package ap

import (
	"strconv"
	"strings"
	"sync/atomic"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/filters"
)

var cnt = atomic.Int64{}

func baseIRI(iri vocab.IRI) vocab.IRI {
	ub, err := iri.GetLink().URL()
	if err != nil {
		return iri
	}
	ub.Path = ""
	ub.RawQuery = ""
	ub.Fragment = ""
	return vocab.IRI(ub.String())
}

// GenerateID generates a unique identifier for the 'it' [vocab.Item].
func GenerateID(it vocab.Item, partOf vocab.IRI, by vocab.Item) (vocab.ID, error) {
	typ := it.GetType()
	typLbl := "tag"
	base := baseIRI(partOf)
	if base == "" {
		if base = baseIRI(by.GetID()); base == "" {
			_ = vocab.OnObject(it, func(ob *vocab.Object) error {
				base = baseIRI(ob.AttributedTo.GetLink())
				return nil
			})
		}
	}
	switch {
	case vocab.ActorTypes.Match(typ):
		base = filters.ActorsType.IRI(base)
	case append(vocab.ActivityTypes, vocab.IntransitiveActivityTypes...).Match(typ):
		base = filters.ActivitiesType.IRI(base)
	default:
		base = filters.ObjectsType.IRI(base)
	}

	if typ != nil {
		typLbl = strings.ToLower(typ.AsTypes().String())
	}
	var err error
	switch {
	case vocab.IsLink(it):
		err = vocab.OnLink(it, func(l *vocab.Link) error {
			cnt.Add(1)
			l.ID = base.AddPath(typLbl + "-" + strconv.FormatInt(cnt.Load(), 10))
			return nil
		})
	case vocab.IsObject(it):
		err = vocab.OnObject(it, func(o *vocab.Object) error {
			cnt.Add(1)
			o.ID = base.AddPath(typLbl + "-" + strconv.FormatInt(cnt.Load(), 10))
			return nil
		})
	}
	return it.GetID(), err
}
