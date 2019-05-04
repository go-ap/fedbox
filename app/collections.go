package app

import (
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/fedbox/internal/errors"
	j "github.com/go-ap/jsonld"
	"net/http"
	"strings"
)

func renderCollection(c as.CollectionInterface) ([]byte, error) {
	return j.Marshal(c)
}

// CollectionHandlerFn is the type that we're using to represent handlers that return ActivityStreams
// Collection objects. It needs to implement the http.Handler interface
type CollectionHandlerFn func(http.ResponseWriter, *http.Request) (as.CollectionInterface, error)

// ServeHTTP implements the http.Handler interface for the CollectionHandlerFn type
func (c CollectionHandlerFn) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var dat []byte
	var status int

	if r.Method != http.MethodGet || r.Method != http.MethodHead {
		status = http.StatusNotAcceptable
		dat, _ = errors.Render(errors.MethodNotAllowedf("invalid HTTP method"))
	}

	if col, err := c(w, r); err != nil {
		// HandleError
		status = http.StatusInternalServerError
		dat, _ = errors.Render(err)
	} else {
		if dat, err = renderCollection(col); err != nil {
			status = http.StatusInternalServerError
			dat, _ = errors.Render(err)
		} else {
			status = http.StatusOK
		}
	}

	w.WriteHeader(status)
	if r.Method == http.MethodGet {
		w.Write(dat)
	}
}

// CollectionType
type CollectionType string

const (
	Outbox    = CollectionType("outbox")
	Inbox     = CollectionType("inbox")
	Shares    = CollectionType("shares")
	Replies   = CollectionType("replies") // activitystreams
	Following = CollectionType("following")
	Followers = CollectionType("followers")
	Liked     = CollectionType("liked")
	Likes     = CollectionType("likes")
)

// EndPointTyper allows external packages to tell us which collection the current HTTP request addresses
type EndPointTyper interface {
	Type(r *http.Request) CollectionType
}

var validActivityCollection = []CollectionType{
	Outbox,
	Inbox,
	Likes,
	Shares,
	Replies, // activitystreams
}

// ValidActivityCollection shows if the current ActivityPub end-point type is a valid one for handling Activities
func ValidActivityCollection(typ string) bool {
	for _, t := range validActivityCollection {
		if strings.ToLower(typ) == string(t) {
			return true
		}
	}
	return false
}

var validObjectCollection = []CollectionType{
	Following,
	Followers,
	Liked,
}

// ValidActivityCollection shows if the current ActivityPub end-point type is a valid one for handling Objects
func ValidObjectCollection(typ string) bool {
	for _, t := range validObjectCollection {
		if strings.ToLower(typ) == string(t) {
			return true
		}
	}
	return false
}

func ValidCollection(typ string) bool {
	return ValidActivityCollection(typ) || ValidObjectCollection(typ)
}

// HandleActivityCollection serves content from the outbox, inbox, likes, shares and replies end-points
// that return ActivityPub collections containing activities
func HandleActivityCollection(w http.ResponseWriter, r *http.Request) (as.CollectionInterface, error) {
	return nil, errors.NotImplementedf("not implemented")
}

// HandleObjectCollection serves content from following, followers, liked, and likes end-points
// that return ActivityPub collections containing plain objects
func HandleObjectCollection(w http.ResponseWriter, r *http.Request) (as.CollectionInterface, error) {
	return nil, errors.NotImplementedf("not implemented")
}
